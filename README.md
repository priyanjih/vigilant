# 🛡️ Vigilant

**Vigilant** is a local observability early-warning system that combines Prometheus metrics, log pattern matching, and a lightweight local LLM to detect issues *before* your services catch fire.

Built for resource-constrained environments, burnout-prone teams, and those tired of "reactive monitoring" that only screams after things break.

---

## 🚀 What It Does

- Connects to **Prometheus** and pulls live alerts
- Tracks active services/nodes under risk
- (Coming soon) Parses logs and matches known failure symptoms
- (Coming soon) Predicts likely fallout using rule-based heuristics
- (Coming soon) Summarizes probable root causes using a local LLM (like LLaMA 3, TinyLlama, Mistral, etc.)
- Outputs clean, CLI-readable summaries of what’s breaking, why, and what’s next

---

## 🧱 Current Status

✅ Prometheus alert fetcher (working)  
🔜 TTL-based risk tracker  
🔜 Log pattern filter  
🔜 LLM root cause summarizer  
🔜 Output to CLI, JSON, or Grafana panel

---

## ⚙️ Running Vigilant (Dev Mode)

Make sure Prometheus is running locally on `localhost:9090`.

Then run:

```bash
go run ./cmd/vigilant

