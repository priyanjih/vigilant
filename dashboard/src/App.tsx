import { useEffect, useState } from "react";

interface APISymptom {
  pattern: string;
  count: number;
}

interface APIMetric {
  name: string;
  value: number;
  operator: string;
  threshold: number;
}

interface APIRiskItem {
  service: string;
  alert: string;
  severity: string;
  score: number;
  symptoms: APISymptom[];
  metrics: APIMetric[];
  summary: string;
  risk: string;
}


export default function App() {
  const [data, setData] = useState<APIRiskItem[]>([]);
  const [selected, setSelected] = useState<APIRiskItem | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      const res = await fetch("/api/risks");
      const json = await res.json();
      json.sort((a: APIRiskItem, b: APIRiskItem) => b.score - a.score);
      setData(json);
    };

    fetchData();
    const interval = setInterval(fetchData, 10000); // every 10s
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="grid grid-cols-3 h-screen text-sm text-white bg-zinc-900">
      <div className="col-span-1 overflow-y-auto border-r border-zinc-700 p-4">
        <h2 className="text-xl font-semibold mb-4">⚠️ Risky Services</h2>
        {data.map((item) => (
          <div
            key={item.service}
            onClick={() => setSelected(item)}
            className={`p-2 mb-2 rounded cursor-pointer hover:bg-zinc-700 ${
              selected?.service === item.service ? "bg-zinc-700" : ""
            }`}
          >
          <div className="flex items-center gap-2">
            <span className="text-red-400 font-bold">{item.score}</span>
            <span
              className={`px-2 py-0.5 rounded-full text-xs font-semibold ${
                item.risk === "High"
                  ? "bg-red-600"
                  : item.risk === "Medium"
                  ? "bg-yellow-600"
                  : item.risk === "Low"
                  ? "bg-green-600"
                  : "bg-gray-600"
              }`}
            >
              {item.risk}
            </span>
          </div>
          <div className="text-zinc-400 text-xs">{item.alert}</div>
          </div>
        ))}
      </div>

      <div className="col-span-2 p-6 overflow-y-auto">
        {selected ? (
          <>
            <h2 className="text-xl font-semibold mb-4">
            {selected.service} — Score: {selected.score} — Risk: {selected.risk}
            </h2>

            <div className="mb-4">
              <h3 className="font-semibold text-zinc-300">🚨 Alert</h3>
              <p>{selected.alert} ({selected.severity})</p>
            </div>

            {selected.symptoms?.length > 0 ? (
              <ul className="list-disc ml-6">
                {selected.symptoms.map((s, i) => (
                  <li key={i}>
                    {s.pattern} — matched {s.count} times
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-zinc-400 italic">No symptoms available</p>
            )}
            <div className="mb-4">
              <h3 className="font-semibold text-zinc-300">📊 Metrics</h3>
              {selected.metrics && selected.metrics.length > 0 ? (
                <ul className="list-disc ml-6">
                  {selected.metrics.map((m, i) => (
                    <li key={i}>
                      {m.name}: {m.value.toFixed(2)} {m.operator} {m.threshold}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-zinc-400 italic">No metrics available</p>
              )}
            </div>

            <div className="mb-4">
              <h3 className="font-semibold text-zinc-300">🧠 Summary</h3>
              <pre className="bg-zinc-800 p-3 rounded whitespace-pre-wrap break-words">{selected.summary}</pre>
            </div>
          </>
        ) : (
          <p className="text-zinc-400">Select a service from the left panel</p>
        )}
      </div>
    </div>
  );
}
