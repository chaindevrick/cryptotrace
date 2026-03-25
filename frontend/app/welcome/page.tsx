'use client';

import React from 'react';
import { 
  ShieldAlert, 
  BrainCircuit, 
  Network, 
  Landmark, 
  Container, 
  LayoutTemplate, 
  ServerCog, 
  Database,
  ArrowRight
} from 'lucide-react';

export default function WelcomePage() {
  return (
    <main className="min-h-screen bg-[#0A0A0C] text-slate-200 font-sans overflow-y-auto selection:bg-[#00E0FF] selection:text-black pb-20">
      {/* 背景點陣與光暈 */}
      <div className="fixed inset-0 z-0 bg-[radial-gradient(#ffffff15_1px,transparent_1px)] [background-size:24px_24px] pointer-events-none" />
      <div className="fixed top-[-20%] left-[-10%] w-[50%] h-[50%] bg-[#00E0FF]/5 blur-[120px] rounded-full pointer-events-none" />
      <div className="fixed bottom-[-20%] right-[-10%] w-[40%] h-[40%] bg-[#FF003C]/5 blur-[120px] rounded-full pointer-events-none" />

      <div className="relative z-10 max-w-6xl mx-auto px-6 pt-24">
        
        {/* ================= HERO SECTION ================= */}
        <div className="flex flex-col items-center text-center mb-20 animate-fade-in-up">
          <div className="flex items-center gap-4 mb-6">
            <ShieldAlert className="text-[#00E0FF] drop-shadow-[0_0_15px_rgba(0,224,255,0.5)]" size={56} />
            <h1 className="text-6xl md:text-7xl font-bold tracking-[0.2em] text-white">
              CRYPTO<span className="text-[#00E0FF]">TRACE</span>
            </h1>
          </div>
          <p className="text-slate-400 text-lg md:text-xl font-mono tracking-widest uppercase mb-10 max-w-2xl">
            Enterprise-grade Blockchain AML & Forensics System
          </p>
          <button 
            onClick={() => window.location.href = '/'} // 導向你的主系統頁面
            className="group relative px-8 py-4 bg-[#00E0FF]/10 border border-[#00E0FF]/50 rounded-lg text-[#00E0FF] font-mono tracking-widest text-sm hover:bg-[#00E0FF]/20 hover:shadow-[0_0_30px_rgba(0,224,255,0.3)] transition-all duration-300 flex items-center gap-3 overflow-hidden"
          >
            <span className="relative z-10 flex items-center gap-2">
              LAUNCH TERMINAL <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform" />
            </span>
          </button>
        </div>

        {/* ================= 核心功能 (Features) ================= */}
        <div className="mb-24">
          <div className="flex items-center gap-4 mb-8">
            <div className="h-px bg-gradient-to-r from-transparent to-[#00E0FF]/50 flex-1" />
            <h2 className="text-xs font-mono tracking-[0.3em] text-[#00E0FF] uppercase">Core Features</h2>
            <div className="h-px bg-gradient-to-l from-transparent to-[#00E0FF]/50 flex-1" />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Feature 1 */}
            <div className="bg-[#121216]/80 backdrop-blur-xl border border-white/10 rounded-xl p-8 hover:border-[#FF003C]/50 hover:shadow-[0_0_30px_rgba(255,0,60,0.1)] transition-all group">
              <BrainCircuit className="text-[#FF003C] mb-6" size={36} />
              <h3 className="text-lg font-bold text-white tracking-wider mb-3">孤立森林 AI 異常檢測</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                導入非監督式機器學習與動態時間窗。將資金流轉化為多維度時序特徵，精準捕捉洗錢行為，並提供異常依據。
              </p>
            </div>

            {/* Feature 2 */}
            <div className="bg-[#121216]/80 backdrop-blur-xl border border-white/10 rounded-xl p-8 hover:border-[#00E0FF]/50 hover:shadow-[0_0_30px_rgba(0,224,255,0.1)] transition-all group">
              <Network className="text-[#00E0FF] mb-6" size={36} />
              <h3 className="text-lg font-bold text-white tracking-wider mb-3">拓撲化視覺前端</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                採用 Dagre 由左至右階層佈局與動態色彩繼承，讓複雜的資金流向一目了然。
              </p>
            </div>

            {/* Feature 3 */}
            <div className="bg-[#121216]/80 backdrop-blur-xl border border-white/10 rounded-xl p-8 hover:border-[#00FF9D]/50 hover:shadow-[0_0_30px_rgba(0,255,157,0.1)] transition-all group">
              <Landmark className="text-[#00FF9D] mb-6" size={36} />
              <h3 className="text-lg font-bold text-white tracking-wider mb-3">交易所與跨鏈橋追蹤</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                自動辨識並標註 CEX (中心化交易所)、DEX (去中心化流動性池) 與 Cross-Chain Bridges，直觀呈現實體名稱，有效追蹤資金斷點。
              </p>
            </div>
          </div>
        </div>

        {/* ================= 技術棧 (Tech Stack) ================= */}
        <div>
          <div className="flex items-center gap-4 mb-8">
            <div className="h-px bg-gradient-to-r from-transparent to-slate-500/50 flex-1" />
            <h2 className="text-xs font-mono tracking-[0.3em] text-slate-400 uppercase">Architecture & Tech Stack</h2>
            <div className="h-px bg-gradient-to-l from-transparent to-slate-500/50 flex-1" />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            
            <div className="flex items-center gap-4 bg-white/5 border border-white/5 rounded-lg p-5 hover:bg-white/10 transition-colors">
              <Container className="text-blue-400" size={24} />
              <div>
                <div className="text-sm font-bold text-white tracking-wider">Docker 容器化</div>
                <div className="text-xs text-slate-500 font-mono mt-1">CI/CD Ready</div>
              </div>
            </div>

            <div className="flex items-center gap-4 bg-white/5 border border-white/5 rounded-lg p-5 hover:bg-white/10 transition-colors">
              <LayoutTemplate className="text-cyan-400" size={24} />
              <div>
                <div className="text-sm font-bold text-white tracking-wider">React + Cytoscape</div>
                <div className="text-xs text-slate-500 font-mono mt-1">Frontend UI</div>
              </div>
            </div>

            <div className="flex items-center gap-4 bg-white/5 border border-white/5 rounded-lg p-5 hover:bg-white/10 transition-colors">
              <ServerCog className="text-purple-400" size={24} />
              <div>
                <div className="text-sm font-bold text-white tracking-wider">Go + Python</div>
                <div className="text-xs text-slate-500 font-mono mt-1">API & AI Microservices</div>
              </div>
            </div>

            <div className="flex items-center gap-4 bg-white/5 border border-white/5 rounded-lg p-5 hover:bg-white/10 transition-colors">
              <Database className="text-emerald-400" size={24} />
              <div>
                <div className="text-sm font-bold text-white tracking-wider">PostgreSQL + Dune</div>
                <div className="text-xs text-slate-500 font-mono mt-1">Dynamic Data Lake</div>
              </div>
            </div>

          </div>
        </div>

      </div>
    </main>
  );
}