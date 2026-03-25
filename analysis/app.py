import os
from typing import Dict, Any, List
from flask import Flask, request, jsonify, send_file
import pandas as pd
import numpy as np
from sklearn.ensemble import IsolationForest
import psycopg2
import io
from datetime import datetime

app = Flask(__name__)

LATEST_ANALYSIS_CACHE = {}

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
    start_time = payload.get('startTime', 0)
    end_time = payload.get('endTime', 0)

    if not target_wallet_address:
        return jsonify({"error": "Missing target address"}), 400

    print(f"\n🔍 [AI Engine] Commencing KYT analysis for wallet: {target_wallet_address}", flush=True)

    conn = get_db_connection()
    cursor = conn.cursor()

    try:
        # PHASE 1: Entity Whitelisting
        cursor.execute("SELECT label FROM wallets WHERE address = %s", (target_wallet_address,))
        row = cursor.fetchone()
        entity_label = row[0] if row else 'wallet'

        if entity_label not in ['wallet', 'HighRisk']:
            print(f"🛡️ [AI Engine] Execution halted: Target is a verified entity ({entity_label}).", flush=True)
            return jsonify({"status": "exempt", "anomalies_found": 0, "anomaly_details": []}), 200

        # PHASE 2: Adaptive Baseline Expansion
        MIN_SAMPLES_REQUIRED = 50
        INITIAL_LOOKBACK_DAYS = 30
        MAX_LOOKBACK_DAYS = 90
        
        training_start_time = start_time - (INITIAL_LOOKBACK_DAYS * 24 * 3600) if start_time > 0 else 0

        def fetch_ego_network(t_start, t_end):
            query = """
                WITH RECURSIVE ego_network AS (
                    SELECT %s::varchar AS address, 0 AS depth
                    UNION
                    SELECT CASE WHEN t.from_address = c.address THEN t.to_address ELSE t.from_address END, c.depth + 1
                    FROM transactions t 
                    JOIN ego_network c ON (t.from_address = c.address OR t.to_address = c.address)
                    JOIN wallets w ON c.address = w.address
                    WHERE c.depth < 2 AND (w.label IN ('wallet', 'HighRisk') OR c.depth = 0)
                    AND (%s::bigint = 0 OR t.timestamp >= %s)
                    AND (%s::bigint = 0 OR t.timestamp <= %s)
                )
                SELECT hash, from_address, to_address, amount, timestamp, type 
                FROM transactions 
                WHERE from_address IN (SELECT address FROM ego_network) 
                   OR to_address IN (SELECT address FROM ego_network)
            """
            import warnings
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', UserWarning)
                return pd.read_sql(query, conn, params=(
                    target_wallet_address, t_start, t_start, t_end, t_end
                ))

        local_context_tx_df = fetch_ego_network(training_start_time, end_time)

        if start_time > 0 and len(local_context_tx_df) < MIN_SAMPLES_REQUIRED:
            print(f"⚠️ [AI Engine] Sparse data ({len(local_context_tx_df)} edges). Expanding lookback to {MAX_LOOKBACK_DAYS} days...", flush=True)
            extended_start_time = start_time - (MAX_LOOKBACK_DAYS * 24 * 3600)
            local_context_tx_df = fetch_ego_network(extended_start_time, end_time)

        if local_context_tx_df.empty or len(local_context_tx_df) < 5:
            print(f"🛑 [AI Engine] Insufficient graph density for ML. Halting.", flush=True)
            return jsonify({"status": "insufficient_data", "anomalies_found": 0, "anomaly_details": []}), 200

        # PHASE 3: Feature Engineering
        local_context_tx_df = local_context_tx_df.sort_values(by=['from_address', 'timestamp']).reset_index(drop=True)
        local_context_tx_df['datetime'] = pd.to_datetime(local_context_tx_df['timestamp'], unit='s')
        local_context_tx_df['time_diff'] = local_context_tx_df.groupby('from_address')['timestamp'].diff().fillna(0)
        
        df_time_indexed = local_context_tx_df.set_index('datetime')
        rolling_frequency_series = df_time_indexed.groupby('from_address')['amount'].rolling('24h').count()
        local_context_tx_df['tx_freq_24h'] = rolling_frequency_series.reset_index(level=0, drop=True).values
        
        feature_columns = ['amount', 'time_diff', 'tx_freq_24h']
        baseline_feature_matrix = local_context_tx_df[feature_columns].values 

        # PHASE 4: Unsupervised Learning
        isolation_forest_model = IsolationForest(contamination='auto', random_state=42) 
        isolation_forest_model.fit(baseline_feature_matrix)

        amount_95th = np.percentile(local_context_tx_df['amount'], 95)
        freq_95th = np.percentile(local_context_tx_df['tx_freq_24h'], 95)

        # PHASE 5: Temporal Slicing & XAI Verification
        target_wallet_mask = (local_context_tx_df['from_address'] == target_wallet_address) | (local_context_tx_df['to_address'] == target_wallet_address)
        target_tx_df = local_context_tx_df[target_wallet_mask].copy()

        if start_time > 0:
            target_tx_df = target_tx_df[target_tx_df['timestamp'] >= start_time]
        if end_time > 0:
            target_tx_df = target_tx_df[target_tx_df['timestamp'] <= end_time]
        
        if target_tx_df.empty:
            return jsonify({"status": "no_target_data_in_window", "anomalies_found": 0, "anomaly_details": []}), 200

        target_feature_matrix = target_tx_df[feature_columns].values
        
        target_tx_df['ai_label'] = isolation_forest_model.predict(target_feature_matrix)
        target_tx_df['anomaly_score'] = isolation_forest_model.decision_function(target_feature_matrix) 

        def extract_compliance_reasons(row: pd.Series) -> List[str]:
            reasons = []
            if row['ai_label'] != -1: return reasons
            if row['amount'] < 3000: return reasons 
                
            if row['amount'] > amount_95th:
                reasons.append(f"Volume ({row['amount']:.2f} U) exceeds network 95th percentile")
            if 0 < row['time_diff'] < 60:
                reasons.append(f"Bot-like high-velocity transfer ({int(row['time_diff'])}s)")
            if row['tx_freq_24h'] > freq_95th:
                reasons.append(f"Frequency ({int(row['tx_freq_24h'])} txs/24h) exceeds network 95th percentile")
            if not reasons:
                reasons.append(f"Multi-dimensional structural anomaly (Score: {row['anomaly_score']:.3f})")
            return reasons

        target_tx_df['compliance_reasons'] = target_tx_df.apply(extract_compliance_reasons, axis=1)
        target_tx_df['is_verified_anomaly'] = target_tx_df['compliance_reasons'].apply(lambda x: len(x) > 0)
        verified_anomalies_count = int(target_tx_df['is_verified_anomaly'].sum())
        
        # PHASE 6: State Mutation & Reporting
        compliance_report = []

        if verified_anomalies_count > 0:
            anomalous_transactions = target_tx_df[target_tx_df['is_verified_anomaly']]
            for _, row in anomalous_transactions.iterrows():
                compliance_report.append({
                    "tx_hash": row['hash'],
                    "amount": float(row['amount']),
                    "timestamp": int(row['timestamp']),
                    "reasons": row['compliance_reasons'],
                    "vector": "OUT" if str(row['from_address']).lower() == target_wallet_address else "IN "
                })

            update_risk_query = "UPDATE wallets SET label = 'HighRisk' WHERE address = %s AND label = 'wallet'"
            cursor.execute(update_risk_query, (target_wallet_address,))
            conn.commit()
            print(f"🚨 [AI] Classification Complete: {verified_anomalies_count} illicit signatures detected.", flush=True)

        # 💡 將最新出爐的 AI 分析報告存入快取！
        LATEST_ANALYSIS_CACHE[target_wallet_address] = {
            "timestamp": datetime.utcnow(),
            "anomalies": compliance_report,
            "baseline_size": len(local_context_tx_df),
            "target_txs_count": len(target_tx_df)
        }

        return jsonify({
            "status": "analyzed", 
            "anomalies_found": verified_anomalies_count,
            "anomaly_details": compliance_report 
        }), 200

    except Exception as e:
        return jsonify({"error": "Internal AI Engine Failure", "details": str(e)}), 500
    finally:
        cursor.close()
        conn.close()

# =====================================================================
# 📊 匯出報告 API：直接從 AI 記憶體快取讀取 XAI 異常理由
# =====================================================================
@app.route('/export_report', methods=['GET'])
def export_report():
    target_wallet = request.args.get('address', '').lower()
    if not target_wallet:
        return jsonify({"error": "Missing target address"}), 400

    # 💡 從記憶體拿出剛剛算好的 AI 報告
    ai_data = LATEST_ANALYSIS_CACHE.get(target_wallet)

    report_lines = []
    report_lines.append(f"# 🛡️ CryptoTrace AML Intelligence Report")
    report_lines.append(f"> **Generated (UTC):** `{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')}`")
    report_lines.append(f"> **Target Entity:** `{target_wallet}`\n")
    
    if not ai_data:
        report_lines.append("⚠️ **Notice:** No recent AI analysis found in memory. Please run the analysis on the dashboard first.")
    else:
        anomalies = ai_data["anomalies"]
        baseline_size = ai_data["baseline_size"]
        target_txs_count = ai_data["target_txs_count"]

        report_lines.append(f"## 1. Executive Summary")
        report_lines.append(f"- **Network Baseline Context:** `{baseline_size}` transactions analyzed")
        report_lines.append(f"- **Target Window Activity:** `{target_txs_count}` transactions")
        report_lines.append(f"- **Verified ML Anomalies:** `{len(anomalies)}` critical signatures detected\n")

        report_lines.append(f"## 2. AI Detected Anomalies (XAI Insights)")
        
        if len(anomalies) == 0:
            report_lines.append("✅ *Behavioral distribution is normal. No critical structural anomalies detected by Isolation Forest.*")
        else:
            # 💡 這裡將真實的 AI 觸發理由寫進表格！
            report_lines.append("| Timestamp (UTC) | Transaction Hash | Vector | Amount | AI ML Triggers |")
            report_lines.append("|-----------------|------------------|--------|--------|----------------|")
            
            for item in anomalies:
                dt_str = datetime.utcfromtimestamp(item['timestamp']).strftime('%Y-%m-%d %H:%M:%S')
                hsh = item['tx_hash']
                amt = item['amount']
                vec = item['vector']
                
                # 將多個理由用 Markdown 的 <br> 換行符號連接
                reasons_html = "<br> • ".join([""] + item['reasons']).strip()
                
                report_lines.append(f"| {dt_str} | `{hsh}...` | {vec} | **{amt:,.2f} U** | {reasons_html} |")

    report_lines.append(f"\n## 3. System Metadata")
    report_lines.append("- **AI Engine:** `CryptoTrace Isolation Forest v1.2`")
    report_lines.append("- **Feature Vectors:** `[Amount, TimeDelta, RollingFreq_24h]`")
    report_lines.append("- **Compliance Status:** *Pending Human Review*")

    markdown_content = "\n".join(report_lines)
    
    mem = io.BytesIO()
    mem.write(markdown_content.encode('utf-8'))
    mem.seek(0)
    
    return send_file(
        mem,
        mimetype='text/markdown',
        as_attachment=True,
        download_name=f"CryptoTrace_Report_{target_wallet[:8]}.md"
    )

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)