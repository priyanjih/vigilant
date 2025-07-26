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
  confidence: number;
  root_cause: string;
  immediate_actions: string[];
  investigation_steps: string[];
  prevention: string;
  timestamp: string;
}


export default function App() {
  const [data, setData] = useState<APIRiskItem[]>([]);
  const [selected, setSelected] = useState<APIRiskItem | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');

  useEffect(() => {
    let ws: WebSocket;
    let reconnectTimeout: number;
    let fallbackInterval: number;

    const fetchDataFallback = async () => {
      try {
        const res = await fetch("/api/risks");
        const json = await res.json();
        json.sort((a: APIRiskItem, b: APIRiskItem) => b.score - a.score);
        setData(json);
      } catch (error) {
        console.error('REST API fallback failed:', error);
      }
    };

    const connectWebSocket = () => {
      // Close existing connection if any
      if (ws && ws.readyState !== WebSocket.CLOSED) {
        ws.close();
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/ws`;
      
      console.log('Attempting WebSocket connection to:', wsUrl);
      console.log('Current location:', window.location.href);
      setConnectionStatus('connecting');
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected successfully');
        setConnectionStatus('connected');
        // Clear any fallback polling when WebSocket connects
        if (fallbackInterval) {
          clearInterval(fallbackInterval);
          fallbackInterval = 0;
        }
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('WebSocket message received:', message);
          if (message.type === 'risks_update') {
            const sortedData = [...message.data].sort((a: APIRiskItem, b: APIRiskItem) => b.score - a.score);
            setData(sortedData);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected - Code:', event.code, 'Reason:', event.reason, 'WasClean:', event.wasClean);
        setConnectionStatus('disconnected');
        
        // Start fallback polling when WebSocket disconnects
        console.log('Starting REST API fallback polling...');
        fallbackInterval = setInterval(fetchDataFallback, 5000);
        
        // Reconnect after 3 seconds
        reconnectTimeout = setTimeout(() => {
          console.log('Attempting to reconnect WebSocket...');
          if (ws.readyState === WebSocket.CLOSED) {
            connectWebSocket();
          }
        }, 3000);
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        console.error('WebSocket ready state:', ws.readyState);
        setConnectionStatus('disconnected');
        // Start fallback polling on error
        if (!fallbackInterval) {
          console.log('Starting REST API fallback polling due to WebSocket error...');
          fallbackInterval = setInterval(fetchDataFallback, 5000);
        }
      };
    };

    // Initial data fetch
    fetchDataFallback();
    
    // Initial WebSocket connection
    connectWebSocket();

    // Cleanup on unmount
    return () => {
      clearTimeout(reconnectTimeout);
      clearInterval(fallbackInterval);
      if (ws) {
        ws.close();
      }
    };
  }, []);

  const getRiskColor = (risk: string) => {
    switch (risk.toLowerCase()) {
      case "critical": return "bg-red-700 text-red-100";
      case "high": return "bg-red-600 text-red-100";
      case "medium": return "bg-yellow-600 text-yellow-100";
      case "low": return "bg-green-600 text-green-100";
      default: return "bg-gray-600 text-gray-100";
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case "critical": return "text-red-400";
      case "warning": return "text-yellow-400";
      case "info": return "text-blue-400";
      default: return "text-gray-400";
    }
  };

  return (
    <div className="h-screen text-sm text-white bg-zinc-900">
      {/* Header */}
      <div className="border-b border-zinc-700 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-white flex items-center gap-3">
              🔍 <span className="text-blue-400">Vigilant</span> 
              <span className="text-zinc-400 text-lg">AI-Powered Observability</span>
            </h1>
            <p className="text-zinc-400 text-sm mt-1">
              Real-time service monitoring with intelligent root cause analysis
            </p>
          </div>
          
          {/* Connection Status */}
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${
              connectionStatus === 'connected' ? 'bg-green-400' :
              connectionStatus === 'connecting' ? 'bg-yellow-400' : 'bg-red-400'
            }`}></div>
            <span className={`text-xs font-medium ${
              connectionStatus === 'connected' ? 'text-green-400' :
              connectionStatus === 'connecting' ? 'text-yellow-400' : 'text-red-400'
            }`}>
              {connectionStatus === 'connected' ? 'Live' :
               connectionStatus === 'connecting' ? 'Connecting...' : 'Disconnected'}
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 h-[calc(100vh-88px)]">
        {/* Services List */}
        <div className="col-span-1 overflow-y-auto border-r border-zinc-700 p-4">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">🚨 Active Incidents</h2>
            <span className="text-xs text-zinc-400 bg-zinc-800 px-2 py-1 rounded">
              {data.length} services
            </span>
          </div>
          
          {data.length === 0 ? (
            <div className="text-center text-zinc-400 mt-8">
              <div className="text-4xl mb-2">✅</div>
              <p>No active incidents</p>
              <p className="text-xs mt-1">All systems operating normally</p>
            </div>
          ) : (
            data.map((item) => (
              <div
                key={item.service}
                onClick={() => setSelected(item)}
                className={`p-3 mb-3 rounded-lg cursor-pointer hover:bg-zinc-800 transition-colors border ${
                  selected?.service === item.service 
                    ? "bg-zinc-800 border-blue-500" 
                    : "border-zinc-700"
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium text-white truncate">{item.service}</span>
                  <span className={`px-2 py-1 rounded text-xs font-semibold ${getRiskColor(item.risk)}`}>
                    {item.risk}
                  </span>
                </div>
                
                <div className="flex items-center gap-2 mb-1">
                  <span className={`text-xs font-medium ${getSeverityColor(item.severity)}`}>
                    ● {item.severity.toUpperCase()}
                  </span>
                  {item.confidence > 0 && (
                    <span className="text-xs text-zinc-400">
                      {Math.round(item.confidence * 100)}% confident
                    </span>
                  )}
                </div>
                
                <div className="text-xs text-zinc-400 truncate">{item.alert}</div>
                
                {item.symptoms?.length > 0 && (
                  <div className="flex items-center gap-1 mt-2">
                    <span className="text-xs text-orange-400">🔍</span>
                    <span className="text-xs text-zinc-400">
                      {item.symptoms.length} symptoms
                    </span>
                  </div>
                )}
              </div>
            ))
          )}
        </div>

        {/* Detail Panel */}
        <div className="col-span-2 p-6 overflow-y-auto">
          {selected ? (
            <>
              {/* Header */}
              <div className="flex items-center justify-between mb-6 pb-4 border-b border-zinc-700">
                <div>
                  <h2 className="text-2xl font-bold text-white flex items-center gap-3">
                    <span className="text-blue-400">📊</span>
                    {selected.service}
                  </h2>
                  <div className="flex items-center gap-4 mt-2">
                    <span className={`px-3 py-1 rounded-lg text-sm font-semibold ${getRiskColor(selected.risk)}`}>
                      {selected.risk} Risk
                    </span>
                    {selected.confidence > 0 && (
                      <span className="text-sm text-zinc-400">
                        {Math.round(selected.confidence * 100)}% confidence
                      </span>
                    )}
                    <span className="text-xs text-zinc-500">{selected.timestamp}</span>
                  </div>
                </div>
              </div>

              {/* Alert Info */}
              <div className="mb-6 bg-zinc-800 rounded-lg p-4 border border-zinc-700">
                <h3 className="font-semibold text-yellow-400 mb-2 flex items-center gap-2">
                  🚨 Alert Details
                </h3>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <span className="text-zinc-400 text-sm">Alert Name:</span>
                    <p className="text-white">{selected.alert}</p>
                  </div>
                  <div>
                    <span className="text-zinc-400 text-sm">Severity:</span>
                    <p className={`font-medium ${getSeverityColor(selected.severity)}`}>
                      {selected.severity.toUpperCase()}
                    </p>
                  </div>
                </div>
              </div>

              {/* Root Cause Analysis */}
              {selected.root_cause && (
                <div className="mb-6 bg-red-950 border border-red-800 rounded-lg p-4">
                  <h3 className="font-semibold text-red-400 mb-3 flex items-center gap-2">
                    🔍 Root Cause Analysis
                  </h3>
                  <p className="text-red-100 leading-relaxed">{selected.root_cause}</p>
                </div>
              )}

              {/* Immediate Actions */}
              {selected.immediate_actions?.length > 0 && (
                <div className="mb-6 bg-orange-950 border border-orange-800 rounded-lg p-4">
                  <h3 className="font-semibold text-orange-400 mb-3 flex items-center gap-2">
                    ⚡ Immediate Actions Required
                  </h3>
                  <ul className="space-y-2">
                    {selected.immediate_actions.map((action, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <span className="text-orange-400 mt-1">•</span>
                        <code className="text-orange-100 text-sm bg-black bg-opacity-30 px-2 py-1 rounded flex-1">
                          {action}
                        </code>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
                {/* Symptoms */}
                <div className="bg-zinc-800 rounded-lg p-4 border border-zinc-700">
                  <h3 className="font-semibold text-blue-400 mb-3 flex items-center gap-2">
                    🩺 Log Symptoms
                  </h3>
                  {selected.symptoms?.length > 0 ? (
                    <ul className="space-y-2">
                      {selected.symptoms.map((s, i) => (
                        <li key={i} className="flex items-center justify-between">
                          <span className="text-zinc-300">{s.pattern}</span>
                          <span className="text-red-400 font-mono text-sm bg-red-900 px-2 py-1 rounded">
                            {s.count}x
                          </span>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-zinc-500 italic">No log symptoms detected</p>
                  )}
                </div>

                {/* Metrics */}
                <div className="bg-zinc-800 rounded-lg p-4 border border-zinc-700">
                  <h3 className="font-semibold text-green-400 mb-3 flex items-center gap-2">
                    📈 Triggered Metrics
                  </h3>
                  {selected.metrics?.length > 0 ? (
                    <ul className="space-y-2">
                      {selected.metrics.map((m, i) => (
                        <li key={i} className="text-sm">
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-zinc-300 font-medium">{m.name}</span>
                            <span className="text-yellow-400 font-mono">
                              {m.value.toFixed(2)} {m.operator} {m.threshold}
                            </span>
                          </div>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-zinc-500 italic">No metrics triggered</p>
                  )}
                </div>
              </div>

              {/* Investigation Steps */}
              {selected.investigation_steps?.length > 0 && (
                <div className="mb-6 bg-blue-950 border border-blue-800 rounded-lg p-4">
                  <h3 className="font-semibold text-blue-400 mb-3 flex items-center gap-2">
                    🔬 Investigation Steps
                  </h3>
                  <ul className="space-y-2">
                    {selected.investigation_steps.map((step, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <span className="text-blue-400 mt-1">{i + 1}.</span>
                        <code className="text-blue-100 text-sm bg-black bg-opacity-30 px-2 py-1 rounded flex-1">
                          {step}
                        </code>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Prevention */}
              {selected.prevention && (
                <div className="mb-6 bg-green-950 border border-green-800 rounded-lg p-4">
                  <h3 className="font-semibold text-green-400 mb-3 flex items-center gap-2">
                    🛡️ Prevention Strategy
                  </h3>
                  <p className="text-green-100 leading-relaxed">{selected.prevention}</p>
                </div>
              )}

            </>
          ) : (
            <div className="flex items-center justify-center h-full text-center">
              <div>
                <div className="text-6xl mb-4">🔍</div>
                <h3 className="text-xl text-zinc-400 mb-2">No incident selected</h3>
                <p className="text-zinc-500">Select a service from the left panel to view detailed analysis</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
