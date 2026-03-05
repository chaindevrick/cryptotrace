import os
from flask import Flask, request, jsonify
import pandas as pd
import numpy as np
from sklearn.ensemble import IsolationForest
import psycopg2

app = Flask(__name__)

def get_db_connection():
    db_url = os.getenv("DATABASE_URL", "postgresql://postgres:password123@postgres:5432/cryptotrace")
    return psycopg2.connect(db_url)

@app.route('/analyze', methods=['POST'])
def analyze():
    data = request.json
    target_address = data.get('address').lower()

    print(f"\n🔍 [AI Engine] 收到分析請求，目標錢包: {target_address}", flush=True)

    conn = get_db_connection()
    cursor = conn.cursor()

    try:
        # ==========================================
        # 🛡️ 1. 白名單豁免機制
        # ==========================================
        cursor.execute("SELECT label FROM wallets WHERE address = %s", (target_address,))
        row = cursor.fetchone()
        wallet_label = row[0] if row else 'wallet'

        if wallet_label not in ['wallet', 'HighRisk']:
            print(f"🛡️ [AI Engine] 目標為已知機構或合約 ({wallet_label})，跳過 AI 檢測。", flush=True)
            return jsonify({"status": "exempt", "anomalies_found": 0})

        # ==========================================
        # 🕸️ 2. 獲取局部網路基準資料 (加入超級節點防爆機制！)
        # ==========================================
        query = """
            WITH RECURSIVE ego_network AS (
                SELECT %s::varchar AS address, 0 AS depth
                UNION
                SELECT 
                    CASE WHEN t.from_address = c.address THEN t.to_address ELSE t.from_address END, 
                    c.depth + 1
                FROM transactions t 
                JOIN ego_network c ON (t.from_address = c.address OR t.to_address = c.address)
                JOIN wallets w ON c.address = w.address
                -- 🛑 核心防爆機制：遇到交易所就停止擴散，並且深度降為 2 層以保證效能
                WHERE c.depth < 2 AND (w.label IN ('wallet', 'HighRisk') OR c.depth = 0)
            )
            SELECT from_address, to_address, amount, timestamp, type 
            FROM transactions 
            WHERE from_address IN (SELECT address FROM ego_network) 
               OR to_address IN (SELECT address FROM ego_network)
        """

        # ⚠️ 修正：只傳入一個參數
        df = pd.read_sql(query, conn, params=(target_address,))

        if df.empty or len(df) < 5:
            print(f"⚠️ [AI Engine] 資料量不足 ({len(df)} 筆)，跳過機器學習分析。", flush=True)
            return jsonify({"status": "insufficient_data", "anomalies_found": 0})

        # ==========================================
        # 🌟 3. 特徵工程 (Feature Engineering: 金額 + 時間 + 頻率)
        # ==========================================
        # 必須先按「發送者」與「時間」進行雙重排序，保證後續運算的準確性
        df = df.sort_values(by=['from_address', 'timestamp']).reset_index(drop=True)
        df['datetime'] = pd.to_datetime(df['timestamp'], unit='s')

        # [特徵 A] 時間差：只計算「同一個發送者」與自己上一筆交易的間隔 (秒)
        df['time_diff'] = df.groupby('from_address')['timestamp'].diff().fillna(0)

        # [特徵 B] 頻率：只計算「同一個發送者」過去 24 小時內的交易次數
        df_indexed = df.set_index('datetime')
        freq_series = df_indexed.groupby('from_address')['amount'].rolling('24h').count()
        
        # ✨ 你的完美修正：將第一層的 from_address index 拔除，確保對齊賦值
        df['tx_freq_24h'] = freq_series.reset_index(level=0, drop=True).values
        
        # 定義多維度輸入特徵 X
        features = ['amount', 'time_diff', 'tx_freq_24h']
        X_baseline = df[features].values 
        
        # 計算局部網路的日常交易中位數，用於後續比對
        median_val = df['amount'].median()

        # ==========================================
        # 🌲 4. 機器學習：訓練局部生態的孤立森林
        # ==========================================
        clf = IsolationForest(contamination=0.05, random_state=42) 
        clf.fit(X_baseline)

        # ==========================================
        # 🎯 5. 異常檢測與雙重確認邏輯
        # ==========================================
        target_mask = (df['from_address'] == target_address) | (df['to_address'] == target_address)
        df_target = df[target_mask].copy()
        
        if df_target.empty:
            return jsonify({"status": "no_target_data", "anomalies_found": 0})

        X_target = df_target[features].values
        df_target['ai_label'] = clf.predict(X_target)
        df_target['anomaly_score'] = clf.decision_function(X_target) 

        def is_true_anomaly(row):
            if row['ai_label'] != -1: return False
            if row['amount'] < 3000: return False # 法規絕對金額門檻
                
            is_amount_spike = (row['amount'] > median_val * 5)               
            is_rapid_tx = (row['time_diff'] < 60 and row['time_diff'] > 0)   
            is_high_freq = (row['tx_freq_24h'] > 20)                         
            
            return is_amount_spike or is_rapid_tx or is_high_freq

        df_target['is_true_anomaly'] = df_target.apply(is_true_anomaly, axis=1)
        anomalies_found = int(df_target['is_true_anomaly'].sum())
        
        # ==========================================
        # 🚨 6. 警報與狀態更新
        # ==========================================
        if anomalies_found > 0:
            update_query = "UPDATE wallets SET label = 'HighRisk' WHERE address = %s AND label = 'wallet'"
            cursor.execute(update_query, (target_address,))
            conn.commit()
            print(f"🚨 [AI] 發現 {anomalies_found} 筆異常！已將 {target_address} 標記為 HighRisk。", flush=True)
        else:
            print(f"✅ [AI] 分析完成，行為符合常態 (0/{len(df_target)})。", flush=True)

        return jsonify({
            "status": "analyzed", 
            "network_baseline_txs": len(df),
            "target_txs_analyzed": len(df_target),
            "anomalies_found": anomalies_found
        })

    except Exception as e:
        print(f"❌ [AI Engine] 發生嚴重錯誤: {e}", flush=True)
        return jsonify({"error": str(e)}), 500

    finally:
        # 🧹 絕對防禦：無論發生什麼事，保證關閉連線，釋放資料庫資源！
        cursor.close()
        conn.close()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)