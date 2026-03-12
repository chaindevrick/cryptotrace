import os
from typing import Dict, Any, List
from flask import Flask, request, jsonify
import pandas as pd
import numpy as np
from sklearn.ensemble import IsolationForest
import psycopg2

app = Flask(__name__)

def get_db_connection() -> psycopg2.extensions.connection:
    db_host = os.getenv("DB_HOST", "postgres") 
    db_port = os.getenv("DB_PORT", "5432")
    db_user = os.getenv("DB_USER", "postgres")
    db_password = os.getenv("DB_PASSWORD", "password123")
    db_name = os.getenv("DB_NAME", "cryptotrace")

    print(f"🔌 [AI Engine] Initializing PostgreSQL connection (Host: {db_host}, User: {db_user})...", flush=True)

    return psycopg2.connect(
        host=db_host,
        port=db_port,
        user=db_user,
        password=db_password,
        dbname=db_name
    )

@app.route('/', methods=['GET'])
def health_check() -> tuple[Dict[str, str], int]:
    return jsonify({"status": "healthy", "service": "CryptoTrace AI Engine"}), 200

@app.route('/analyze', methods=['POST'])
def analyze_wallet_behavior() -> tuple[Dict[str, Any], int]:
    payload = request.json
    target_wallet_address = payload.get('address', '').lower()

    if not target_wallet_address:
        return jsonify({"error": "Missing target address"}), 400

    print(f"\n🔍 [AI Engine] Commencing KYT analysis for wallet: {target_wallet_address}", flush=True)

    conn = get_db_connection()
    cursor = conn.cursor()

    try:
        # =====================================================================
        # PHASE 1 & 2: Entity Whitelisting & Local Context Retrieval
        # =====================================================================
        cursor.execute("SELECT label FROM wallets WHERE address = %s", (target_wallet_address,))
        row = cursor.fetchone()
        entity_label = row[0] if row else 'wallet'

        if entity_label not in ['wallet', 'HighRisk']:
            print(f"🛡️ [AI Engine] Execution halted: Target is a verified entity ({entity_label}).", flush=True)
            return jsonify({"status": "exempt", "anomalies_found": 0, "anomaly_details": []}), 200

        ego_network_query = """
            WITH RECURSIVE ego_network AS (
                SELECT %s::varchar AS address, 0 AS depth
                UNION
                SELECT 
                    CASE WHEN t.from_address = c.address THEN t.to_address ELSE t.from_address END, 
                    c.depth + 1
                FROM transactions t 
                JOIN ego_network c ON (t.from_address = c.address OR t.to_address = c.address)
                JOIN wallets w ON c.address = w.address
                WHERE c.depth < 2 AND (w.label IN ('wallet', 'HighRisk') OR c.depth = 0)
            )
            SELECT hash, from_address, to_address, amount, timestamp, type 
            FROM transactions 
            WHERE from_address IN (SELECT address FROM ego_network) 
               OR to_address IN (SELECT address FROM ego_network)
        """

        local_context_tx_df = pd.read_sql(ego_network_query, conn, params=(target_wallet_address,))

        if local_context_tx_df.empty or len(local_context_tx_df) < 5:
            return jsonify({"status": "insufficient_data", "anomalies_found": 0, "anomaly_details": []}), 200

        # =====================================================================
        # PHASE 3: Feature Engineering 
        # =====================================================================
        local_context_tx_df = local_context_tx_df.sort_values(by=['from_address', 'timestamp']).reset_index(drop=True)
        local_context_tx_df['datetime'] = pd.to_datetime(local_context_tx_df['timestamp'], unit='s')
        local_context_tx_df['time_diff'] = local_context_tx_df.groupby('from_address')['timestamp'].diff().fillna(0)
        
        df_time_indexed = local_context_tx_df.set_index('datetime')
        rolling_frequency_series = df_time_indexed.groupby('from_address')['amount'].rolling('24h').count()
        local_context_tx_df['tx_freq_24h'] = rolling_frequency_series.reset_index(level=0, drop=True).values
        
        feature_columns = ['amount', 'time_diff', 'tx_freq_24h']
        baseline_feature_matrix = local_context_tx_df[feature_columns].values 

        # =====================================================================
        # PHASE 4: Unsupervised Learning (Isolation Forest)
        # =====================================================================
        isolation_forest_model = IsolationForest(contamination='auto', random_state=42) 
        isolation_forest_model.fit(baseline_feature_matrix)

        # =====================================================================
        # PHASE 5: Explainable AI (XAI) - Model-Driven Anomaly Detection
        # Design Decision: 信任 ML 模型，捨棄人工寫死的硬性邊界 (Hardcoded Thresholds)。
        # Why: Isolation Forest 會自動在高維度空間 (金額 x 時間差 x 頻率) 找出孤立點。
        #      我們不再用 "amount > 5x median" 來決定異常，而是將 ML 標記 (-1) 視為唯一真理。
        #      底下的邏輯純粹是為了產生「法遵報告 (Compliance Report)」，
        #      利用 95 百分位數 (95th Percentile) 向人類解釋模型為什麼抓出這筆交易。
        # =====================================================================
        target_wallet_mask = (local_context_tx_df['from_address'] == target_wallet_address) | (local_context_tx_df['to_address'] == target_wallet_address)
        target_tx_df = local_context_tx_df[target_wallet_mask].copy()
        
        if target_tx_df.empty:
            return jsonify({"status": "no_target_data", "anomalies_found": 0, "anomaly_details": []}), 200

        target_feature_matrix = target_tx_df[feature_columns].values
        
        # 👑 唯一的判斷標準：讓 ML 模型說了算
        target_tx_df['ai_label'] = isolation_forest_model.predict(target_feature_matrix)
        target_tx_df['anomaly_score'] = isolation_forest_model.decision_function(target_feature_matrix) 

        # 計算局部生態的統計動態天花板，專供 XAI 報表翻譯使用
        amount_95th = np.percentile(local_context_tx_df['amount'], 95)
        freq_95th = np.percentile(local_context_tx_df['tx_freq_24h'], 95)

        def extract_compliance_reasons(row: pd.Series) -> List[str]:
            reasons = []
            
            # 第一道防線：模型沒有標記為 Outlier (-1)，直接放行
            if row['ai_label'] != -1: 
                return reasons
            
            # 灰塵過濾 (Dusting Filter)：即使模型覺得異常，但低於 3000 U 的微小雜訊仍不予起訴
            if row['amount'] < 3000: 
                return reasons 
                
            # XAI 翻譯層：模型判定異常了，我們來告訴法遵人員「模型可能看到了什麼特徵」
            if row['amount'] > amount_95th:
                reasons.append(f"ML Insight: Volume ({row['amount']:.2f} U) exceeds network 95th percentile")
            
            if 0 < row['time_diff'] < 60:
                reasons.append(f"ML Insight: Bot-like high-velocity transfer ({int(row['time_diff'])}s)")
                
            if row['tx_freq_24h'] > freq_95th:
                reasons.append(f"ML Insight: Frequency ({int(row['tx_freq_24h'])} txs/24h) exceeds network 95th percentile")
            
            # 邊緣情況：如果模型標記了，但上述單一維度都沒突破 95%，代表這是多維度結構性異常
            if not reasons:
                reasons.append(f"ML Insight: Multi-dimensional structural anomaly (Score: {row['anomaly_score']:.3f})")

            return reasons

        target_tx_df['compliance_reasons'] = target_tx_df.apply(extract_compliance_reasons, axis=1)
        
        # 只要 reasons 裡有資料，就確認為可起訴的異常
        target_tx_df['is_verified_anomaly'] = target_tx_df['compliance_reasons'].apply(lambda x: len(x) > 0)
        verified_anomalies_count = int(target_tx_df['is_verified_anomaly'].sum())
        
        # =====================================================================
        # PHASE 6: State Mutation & Reporting
        # =====================================================================
        compliance_report = []

        if verified_anomalies_count > 0:
            anomalous_transactions = target_tx_df[target_tx_df['is_verified_anomaly']]
            
            for _, row in anomalous_transactions.iterrows():
                compliance_report.append({
                    "tx_hash": row['hash'],
                    "amount": float(row['amount']),
                    "timestamp": int(row['timestamp']),
                    "reasons": row['compliance_reasons']
                })

            update_risk_query = "UPDATE wallets SET label = 'HighRisk' WHERE address = %s AND label = 'wallet'"
            cursor.execute(update_risk_query, (target_wallet_address,))
            conn.commit()
            
            print(f"🚨 [AI] Classification Complete: {verified_anomalies_count} illicit signatures detected. Entity {target_wallet_address} marked as HighRisk.", flush=True)
            for detail in compliance_report:
                reason_str = " | ".join(detail['reasons'])
                print(f"   👉 Tx: {detail['tx_hash'][:12]}... | Amount: {detail['amount']:,.2f} U | Triggers: {reason_str}", flush=True)
        else:
            print(f"✅ [AI] Classification Complete: Normal behavioral distribution (0/{len(target_tx_df)} anomalies).", flush=True)

        return jsonify({
            "status": "analyzed", 
            "network_baseline_txs": len(local_context_tx_df),
            "target_txs_analyzed": len(target_tx_df),
            "anomalies_found": verified_anomalies_count,
            "anomaly_details": compliance_report 
        }), 200

    except Exception as e:
        print(f"❌ [AI Engine] Critical failure during ML pipeline execution: {e}", flush=True)
        return jsonify({"error": "Internal AI Engine Failure", "details": str(e)}), 500

    finally:
        cursor.close()
        conn.close()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)