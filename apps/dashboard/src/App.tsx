import { useState, useEffect } from 'react';
import { 
  Activity, 
  DollarSign, 
  Wifi, 
  Users, 
  LogOut, 
  Lock, 
  User, 
  RefreshCw, 
  Database,
  Smartphone, 
  Clock, 
  AlertCircle,
  Settings,
  XCircle,
  CheckCircle2,
  TrendingUp,
  Layers
} from 'lucide-react';

// Configuration
const API_BASE = window.location.origin.includes('3000') 
  ? 'http://localhost:8080' 
  : window.location.origin;

interface Session {
  id: number;
  device_id: number;
  device: {
    mac_address: string;
    ip_address: string;
    manufacturer: string;
    device_type: string;
  };
  plan: {
    name: string;
    price_kes: number;
    duration_minutes: number;
    bandwidth_limit_down: number;
    bandwidth_limit_up: number;
  };
  start_time: string;
  end_time: string;
  status: string;
  ip_address: string;
  bytes_in: number;
  bytes_out: number;
}

interface Payment {
  id: number;
  transaction_id: string | null;
  checkout_request_id: string;
  amount_kes: number;
  phone_number: string;
  status: string;
  created_at: string;
}

interface Analytics {
  revenue_total: number;
  revenue_today: number;
  active_users: number;
  total_devices: number;
  bandwidth_usage_mbps: number;
  active_sessions: Session[];
  recent_payments: Payment[];
}

export default function App() {
  const [token, setToken] = useState<string | null>(localStorage.getItem('hotspotos_token'));
  const [username, setUsername] = useState<string>('admin');
  const [password, setPassword] = useState<string>('');
  const [loginError, setLoginError] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [activeTab, setActiveTab] = useState<'dashboard' | 'sessions' | 'payments' | 'plans' | 'settings'>('dashboard');

  const [plans, setPlans] = useState<any[]>([]);
  const [isPlanModalOpen, setIsPlanModalOpen] = useState<boolean>(false);
  const [editingPlan, setEditingPlan] = useState<any | null>(null);
  const [planForm, setPlanForm] = useState({
    name: '',
    duration_minutes: 60,
    price_kes: 0,
    bandwidth_limit_down: 2048,
    bandwidth_limit_up: 1024
  });

  // Analytics State
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [error, setError] = useState<string>('');
  const [isRefreshing, setIsRefreshing] = useState<boolean>(false);
  const [wsConnected, setWsConnected] = useState<boolean>(false);

  // Poll intervals
  useEffect(() => {
    if (!token) return;
    
    fetchAnalytics();
    fetchPlans();
    const interval = setInterval(() => {
      fetchAnalytics();
      fetchPlans();
    }, 5000);

    // Setup WebSockets for instant updates
    let ws: WebSocket;
    const connectWs = () => {
      const wsUrl = API_BASE.replace('http', 'ws') + '/ws';
      ws = new WebSocket(wsUrl);
      
      ws.onopen = () => {
        setWsConnected(true);
        setError('');
      };
      
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          logger("WS update received:", data);
          // Trigger immediate data refresh
          fetchAnalytics();
        } catch (e) {
          console.error(e);
        }
      };

      ws.onclose = () => {
        setWsConnected(false);
        // Try reconnecting in 5 seconds
        setTimeout(connectWs, 5000);
      };

      ws.onerror = () => {
        setWsConnected(false);
      };
    };

    connectWs();

    return () => {
      clearInterval(interval);
      if (ws) ws.close();
    };
  }, [token]);

  const fetchAnalytics = async () => {
    if (!token) return;
    setIsRefreshing(true);
    try {
      const res = await fetch(`${API_BASE}/admin/analytics`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (res.status === 401) {
        handleLogout();
        return;
      }

      if (!res.ok) {
        throw new Error("Failed to fetch analytics data");
      }

      const data = await res.json();
      setAnalytics(data);
      setError('');
    } catch (err: any) {
      setError(err.message || 'Error loading dashboard data');
    } finally {
      setIsRefreshing(false);
    }
  };

  const fetchPlans = async () => {
    if (!token) return;
    try {
      const res = await fetch(`${API_BASE}/plans`);
      if (res.ok) {
        const data = await res.json();
        setPlans(data);
      }
    } catch (err) {
      console.error("Failed to fetch plans", err);
    }
  };

  const handleSavePlan = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const url = editingPlan 
        ? `${API_BASE}/admin/plans/${editingPlan.id}` 
        : `${API_BASE}/admin/plans`;
      const method = editingPlan ? 'PUT' : 'POST';
      
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          name: planForm.name,
          duration_minutes: Number(planForm.duration_minutes),
          price_kes: Number(planForm.price_kes),
          bandwidth_limit_down: Number(planForm.bandwidth_limit_down),
          bandwidth_limit_up: Number(planForm.bandwidth_limit_up)
        })
      });

      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to save plan");
      }

      setIsPlanModalOpen(false);
      setEditingPlan(null);
      setPlanForm({
        name: '',
        duration_minutes: 60,
        price_kes: 0,
        bandwidth_limit_down: 2048,
        bandwidth_limit_up: 1024
      });
      fetchPlans();
    } catch (err: any) {
      alert("Error: " + err.message);
    }
  };

  const handleDeletePlan = async (id: number) => {
    if (!confirm("Are you sure you want to delete this billing package?")) return;
    try {
      const res = await fetch(`${API_BASE}/admin/plans/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to delete plan");
      }
      fetchPlans();
    } catch (err: any) {
      alert("Error: " + err.message);
    }
  };

  const handleLogin = async (e: React.FormEvent) =>>,StartLine:154,TargetContent: {
    e.preventDefault();
    setLoginError('');
    setIsLoading(true);

    try {
      const res = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Login failed');
      }

      localStorage.setItem('hotspotos_token', data.token);
      setToken(data.token);
      setPassword('');
    } catch (err: any) {
      setLoginError(err.message || 'Failed to authenticate');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('hotspotos_token');
    setToken(null);
    setAnalytics(null);
  };

  const handleDisconnect = async (sessionId: number) => {
    if (!confirm("Are you sure you want to disconnect this device and revoke internet access?")) {
      return;
    }

    try {
      const res = await fetch(`${API_BASE}/admin/disconnect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ session_id: sessionId })
      });

      const data = await res.json();
      if (!res.ok) {
        alert("Failed to disconnect: " + (data.error || "Unknown error"));
      } else {
        fetchAnalytics();
      }
    } catch (err: any) {
      alert("Error calling disconnect: " + err.message);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatTime = (timeStr: string) => {
    const d = new Date(timeStr);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  // Helper logger to bypass typescript limits
  const logger = (msg: string, arg: any) => {
    console.log(msg, arg);
  };

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 px-4 relative overflow-hidden">
        {/* Glow Effects */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-orange-600/10 rounded-full blur-[128px]"></div>
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-indigo-600/10 rounded-full blur-[128px]"></div>

        <div className="w-full max-w-md bg-slate-900/80 border border-slate-800 rounded-3xl p-8 backdrop-blur-xl shadow-2xl relative z-10">
          <div className="text-center mb-8">
            <div className="w-16 h-16 bg-gradient-to-tr from-orange-500 to-indigo-600 rounded-2xl mx-auto flex items-center justify-center shadow-lg shadow-orange-500/20 mb-4">
              <Wifi className="w-8 h-8 text-white animate-pulse" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight">HotspotOS Portal</h1>
            <p className="text-slate-400 mt-2 text-sm">Please log in to manage your network nodes</p>
          </div>

          {loginError && (
            <div className="mb-6 p-4 bg-red-500/10 border border-red-500/20 rounded-xl flex items-start gap-3 text-red-400 text-sm">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <span>{loginError}</span>
            </div>
          )}

          <form onSubmit={handleLogin} className="space-y-5">
            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Username</label>
              <div className="relative">
                <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-slate-500">
                  <User className="w-5 h-5" />
                </span>
                <input
                  type="text"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 pl-11 pr-4 text-white placeholder-slate-600 outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                  placeholder="admin"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Password</label>
              <div className="relative">
                <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-slate-500">
                  <Lock className="w-5 h-5" />
                </span>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 pl-11 pr-4 text-white placeholder-slate-600 outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                  placeholder="••••••••"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 text-white font-bold py-3 px-4 rounded-xl shadow-lg shadow-orange-500/25 transition-all duration-200 transform hover:-translate-y-0.5 active:translate-y-0 flex items-center justify-center gap-2"
            >
              {isLoading ? (
                <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin"></div>
              ) : (
                'Sign In'
              )}
            </button>
          </form>

          <div className="mt-8 text-center text-xs text-slate-500">
            <p>HotspotOS Version 1.0.0 (Ubuntu Host)</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex bg-slate-950">
      {/* Sidebar Navigation */}
      <aside className="w-64 border-r border-slate-900 bg-slate-900/20 flex flex-col justify-between shrink-0">
        <div>
          <div className="p-6 border-b border-slate-900 flex items-center gap-3">
            <div className="w-9 h-9 bg-orange-500 rounded-xl flex items-center justify-center font-bold text-white shadow-md shadow-orange-500/20">
              H
            </div>
            <div>
              <h2 className="font-extrabold text-sm tracking-wide text-white">HOTSPOT OS</h2>
              <span className="text-[10px] text-orange-500 font-bold bg-orange-500/10 px-2 py-0.5 rounded-full mt-1 inline-block">LOCAL SERVER</span>
            </div>
          </div>

          <nav className="p-4 space-y-1.5">
            <button
              onClick={() => setActiveTab('dashboard')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'dashboard'
                  ? 'bg-orange-500 text-white shadow-lg shadow-orange-500/15'
                  : 'text-slate-400 hover:text-white hover:bg-slate-900'
              }`}
            >
              <Activity className="w-5 h-5" />
              <span>Overview</span>
            </button>

            <button
              onClick={() => setActiveTab('sessions')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'sessions'
                  ? 'bg-orange-500 text-white shadow-lg shadow-orange-500/15'
                  : 'text-slate-400 hover:text-white hover:bg-slate-900'
              }`}
            >
              <Users className="w-5 h-5" />
              <span>Active Sessions</span>
              {analytics && analytics.active_sessions.length > 0 && (
                <span className="ml-auto bg-slate-950 text-orange-500 text-xs px-2 py-0.5 rounded-full font-extrabold border border-orange-500/20">
                  {analytics.active_sessions.length}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('payments')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'payments'
                  ? 'bg-orange-500 text-white shadow-lg shadow-orange-500/15'
                  : 'text-slate-400 hover:text-white hover:bg-slate-900'
              }`}
            >
              <DollarSign className="w-5 h-5" />
              <span>Payment Logs</span>
            </button>

            <button
              onClick={() => setActiveTab('plans')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'plans'
                  ? 'bg-orange-500 text-white shadow-lg shadow-orange-500/15'
                  : 'text-slate-400 hover:text-white hover:bg-slate-900'
              }`}
            >
              <Layers className="w-5 h-5" />
              <span>Billing Packages</span>
            </button>

            <button
              onClick={() => setActiveTab('settings')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'settings'
                  ? 'bg-orange-500 text-white shadow-lg shadow-orange-500/15'
                  : 'text-slate-400 hover:text-white hover:bg-slate-900'
              }`}
            >
              <Settings className="w-5 h-5" />
              <span>Configuration</span>
            </button>
          </nav>
        </div>

        <div className="p-4 border-t border-slate-900 space-y-4">
          <div className="flex items-center gap-3 px-2">
            <div className={`w-2.5 h-2.5 rounded-full ${wsConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></div>
            <span className="text-xs text-slate-400 font-semibold">{wsConnected ? 'Real-Time Sync Active' : 'Offline / Polling Mode'}</span>
          </div>

          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold text-slate-400 hover:text-red-400 hover:bg-red-500/5 transition-all"
          >
            <LogOut className="w-5 h-5" />
            <span>Sign Out</span>
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto">
        {/* Header */}
        <header className="h-20 border-b border-slate-900 px-8 flex items-center justify-between bg-slate-950/50 backdrop-blur-md sticky top-0 z-10 shrink-0">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-white capitalize">
              {activeTab === 'dashboard' ? 'Overview' : activeTab === 'sessions' ? 'Active Internet Sessions' : activeTab === 'payments' ? 'M-Pesa Payments Audit' : activeTab === 'plans' ? 'Billing Plans & Packages' : 'Settings'}
            </h1>
            <p className="text-xs text-slate-500 mt-1">Real-time telemetry and firewall manager</p>
          </div>

          <div className="flex items-center gap-4">
            <button
              onClick={fetchAnalytics}
              className={`p-2.5 rounded-xl border border-slate-900 bg-slate-900/30 text-slate-400 hover:text-white transition-all ${
                isRefreshing ? 'animate-spin text-orange-500' : ''
              }`}
            >
              <RefreshCw className="w-5 h-5" />
            </button>

            <div className="flex items-center gap-3 pl-4 border-l border-slate-900">
              <div className="w-10 h-10 bg-slate-900 border border-slate-800 rounded-xl flex items-center justify-center text-slate-300 font-bold text-sm">
                AD
              </div>
              <div className="hidden md:block">
                <p className="text-sm font-semibold text-white">System Administrator</p>
                <p className="text-[10px] text-slate-500 uppercase tracking-wider font-extrabold">Superuser Role</p>
              </div>
            </div>
          </div>
        </header>

        {/* Dashboard Content Panels */}
        <div className="p-8 flex-1 max-w-7xl w-full mx-auto space-y-8">
          {error && (
            <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-start gap-3 text-red-400 text-sm">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {activeTab === 'dashboard' && (
            <>
              {/* Stat Cards Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                {/* Card 1: Today's Revenue */}
                <div className="bg-slate-900/30 border border-slate-900 rounded-3xl p-6 relative overflow-hidden group hover:border-orange-500/30 transition-all duration-300">
                  <div className="absolute top-0 right-0 w-32 h-32 bg-orange-500/5 rounded-full blur-3xl group-hover:bg-orange-500/10 transition-all"></div>
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Revenue Today</span>
                      <h3 className="text-3xl font-extrabold text-white mt-3">
                        KES {analytics?.revenue_today.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) || '0.00'}
                      </h3>
                      <p className="text-xs text-green-500 font-semibold flex items-center gap-1 mt-2">
                        <TrendingUp className="w-3.5 h-3.5" />
                        <span>+12.4% vs yesterday</span>
                      </p>
                    </div>
                    <div className="p-3 bg-orange-500/10 border border-orange-500/20 text-orange-500 rounded-2xl">
                      <DollarSign className="w-6 h-6" />
                    </div>
                  </div>
                </div>

                {/* Card 2: Connected Users */}
                <div className="bg-slate-900/30 border border-slate-900 rounded-3xl p-6 relative overflow-hidden group hover:border-indigo-500/30 transition-all duration-300">
                  <div className="absolute top-0 right-0 w-32 h-32 bg-indigo-500/5 rounded-full blur-3xl group-hover:bg-indigo-500/10 transition-all"></div>
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Connected Users</span>
                      <h3 className="text-3xl font-extrabold text-white mt-3">
                        {analytics?.active_users || 0}
                      </h3>
                      <p className="text-xs text-slate-500 font-semibold mt-2">
                        Active firewall leases
                      </p>
                    </div>
                    <div className="p-3 bg-indigo-500/10 border border-indigo-500/20 text-indigo-500 rounded-2xl">
                      <Users className="w-6 h-6" />
                    </div>
                  </div>
                </div>

                {/* Card 3: Bandwidth Usage */}
                <div className="bg-slate-900/30 border border-slate-900 rounded-3xl p-6 relative overflow-hidden group hover:border-blue-500/30 transition-all duration-300">
                  <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/5 rounded-full blur-3xl group-hover:bg-blue-500/10 transition-all"></div>
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Network Bandwidth</span>
                      <h3 className="text-3xl font-extrabold text-white mt-3">
                        {analytics?.bandwidth_usage_mbps.toFixed(1) || '0.0'} <span className="text-lg font-bold text-slate-500">Mbps</span>
                      </h3>
                      <p className="text-xs text-blue-400 font-semibold mt-2">
                        Dynamic rate-limiting set
                      </p>
                    </div>
                    <div className="p-3 bg-blue-500/10 border border-blue-500/20 text-blue-500 rounded-2xl">
                      <Activity className="w-6 h-6" />
                    </div>
                  </div>
                </div>

                {/* Card 4: Total Registered Devices */}
                <div className="bg-slate-900/30 border border-slate-900 rounded-3xl p-6 relative overflow-hidden group hover:border-green-500/30 transition-all duration-300">
                  <div className="absolute top-0 right-0 w-32 h-32 bg-green-500/5 rounded-full blur-3xl group-hover:bg-green-500/10 transition-all"></div>
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Known Devices</span>
                      <h3 className="text-3xl font-extrabold text-white mt-3">
                        {analytics?.total_devices || 0}
                      </h3>
                      <p className="text-xs text-slate-500 font-semibold mt-2">
                        Stored MAC mappings
                      </p>
                    </div>
                    <div className="p-3 bg-green-500/10 border border-green-500/20 text-green-500 rounded-2xl">
                      <Smartphone className="w-6 h-6" />
                    </div>
                  </div>
                </div>
              </div>

              {/* Main Section Layout */}
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* Active Sessions List */}
                <div className="lg:col-span-2 bg-slate-900/20 border border-slate-900 rounded-3xl p-6 space-y-6">
                  <div className="flex justify-between items-center">
                    <div>
                      <h4 className="font-bold text-lg text-white">Active Leases</h4>
                      <p className="text-xs text-slate-500">Live internet connections authorized by firewall</p>
                    </div>
                    <button 
                      onClick={() => setActiveTab('sessions')}
                      className="text-xs font-bold text-orange-500 hover:text-orange-600 transition-all"
                    >
                      View All
                    </button>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                      <thead>
                        <tr className="border-b border-slate-900 text-xs font-semibold uppercase tracking-wider text-slate-500">
                          <th className="py-4">Device Details</th>
                          <th className="py-4">Plan / Speed</th>
                          <th className="py-4">Volume Used</th>
                          <th className="py-4 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-900 text-sm">
                        {analytics?.active_sessions.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="py-8 text-center text-slate-500 font-semibold">
                              No active client leases on the local node.
                            </td>
                          </tr>
                        ) : (
                          analytics?.active_sessions.map((session) => (
                            <tr key={session.id} className="group hover:bg-slate-900/10">
                              <td className="py-4 flex items-center gap-3">
                                <div className="p-2.5 bg-slate-900 border border-slate-800 rounded-xl text-slate-400 group-hover:text-white transition-all">
                                  <Smartphone className="w-5 h-5" />
                                </div>
                                <div>
                                  <p className="font-bold text-white text-xs md:text-sm">{session.device.ip_address}</p>
                                  <p className="text-[10px] text-slate-500 font-mono mt-0.5">{session.device.mac_address}</p>
                                </div>
                              </td>
                              <td className="py-4">
                                <p className="font-semibold text-xs md:text-sm text-slate-200">{session.plan.name}</p>
                                <p className="text-[10px] text-slate-500 mt-0.5">
                                  Limit: {session.plan.bandwidth_limit_down ? `${session.plan.bandwidth_limit_down/1024}M` : '10M'}/{session.plan.bandwidth_limit_up ? `${session.plan.bandwidth_limit_up/1024}M` : '2M'}
                                </p>
                              </td>
                              <td className="py-4">
                                <p className="font-semibold text-xs text-slate-300">↑ {formatBytes(session.bytes_out)}</p>
                                <p className="font-semibold text-xs text-slate-400 mt-0.5">↓ {formatBytes(session.bytes_in)}</p>
                              </td>
                              <td className="py-4 text-right">
                                <button
                                  onClick={() => handleDisconnect(session.id)}
                                  className="px-3 py-1.5 bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white rounded-lg text-xs font-bold transition-all border border-red-500/20"
                                >
                                  Disconnect
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Recent Payments Section */}
                <div className="bg-slate-900/20 border border-slate-900 rounded-3xl p-6 space-y-6">
                  <div className="flex justify-between items-center">
                    <div>
                      <h4 className="font-bold text-lg text-white">Recent Payments</h4>
                      <p className="text-xs text-slate-500">M-Pesa transaction queue status</p>
                    </div>
                  </div>

                  <div className="space-y-4">
                    {analytics?.recent_payments.length === 0 ? (
                      <p className="text-center py-8 text-slate-500 text-sm font-semibold">
                        No transactions registered yet.
                      </p>
                    ) : (
                      analytics?.recent_payments.map((payment) => (
                        <div key={payment.id} className="flex justify-between items-center p-4 bg-slate-900/40 border border-slate-900 rounded-2xl">
                          <div className="flex items-center gap-3">
                            {payment.status === 'completed' ? (
                              <div className="p-2 bg-green-500/10 border border-green-500/20 text-green-500 rounded-xl">
                                <CheckCircle2 className="w-5 h-5" />
                              </div>
                            ) : payment.status === 'pending' ? (
                              <div className="p-2 bg-yellow-500/10 border border-yellow-500/20 text-yellow-500 rounded-xl animate-pulse">
                                <Clock className="w-5 h-5" />
                              </div>
                            ) : (
                              <div className="p-2 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl">
                                <XCircle className="w-5 h-5" />
                              </div>
                            )}
                            <div>
                              <p className="text-sm font-bold text-white">{payment.phone_number}</p>
                              <p className="text-[10px] text-slate-500 font-mono mt-0.5">{payment.transaction_id || payment.checkout_request_id.substring(0, 14) + '...'}</p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className="text-sm font-extrabold text-orange-500">KES {payment.amount_kes}</p>
                            <p className="text-[10px] text-slate-500 mt-0.5">{new Date(payment.created_at).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</p>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>
            </>
          )}

          {activeTab === 'sessions' && (
            <div className="bg-slate-900/20 border border-slate-900 rounded-3xl p-6 space-y-6">
              <div>
                <h4 className="font-bold text-lg text-white">Active Leases & Hardware Mapping</h4>
                <p className="text-xs text-slate-500">List of all active sessions authorized to bypass captive portal redirects</p>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-slate-900 text-xs font-semibold uppercase tracking-wider text-slate-500">
                      <th className="py-4">Client IP</th>
                      <th className="py-4">MAC Address</th>
                      <th className="py-4">Connected Plan</th>
                      <th className="py-4">Usage Volume</th>
                      <th className="py-4">Session Duration</th>
                      <th className="py-4 text-right">Revoke Access</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-900 text-sm">
                    {analytics?.active_sessions.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="py-8 text-center text-slate-500 font-semibold">
                          No active internet sessions currently.
                        </td>
                      </tr>
                    ) : (
                      analytics?.active_sessions.map((session) => (
                        <tr key={session.id} className="hover:bg-slate-900/10">
                          <td className="py-4 font-bold text-white">{session.device.ip_address}</td>
                          <td className="py-4 font-mono text-slate-400 text-xs">{session.device.mac_address}</td>
                          <td className="py-4 font-semibold text-slate-200">{session.plan.name}</td>
                          <td className="py-4">
                            <span className="text-xs text-slate-400">In: {formatBytes(session.bytes_in)}</span>
                            <span className="mx-2 text-slate-600">|</span>
                            <span className="text-xs text-slate-400">Out: {formatBytes(session.bytes_out)}</span>
                          </td>
                          <td className="py-4">
                            <p className="text-xs text-slate-300">Started: {formatTime(session.start_time)}</p>
                            <p className="text-[10px] text-red-400 mt-0.5">Expires: {formatTime(session.end_time)}</p>
                          </td>
                          <td className="py-4 text-right">
                            <button
                              onClick={() => handleDisconnect(session.id)}
                              className="px-3 py-1.5 bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white rounded-lg text-xs font-bold transition-all border border-red-500/20"
                            >
                              Disconnect
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {activeTab === 'payments' && (
            <div className="bg-slate-900/20 border border-slate-900 rounded-3xl p-6 space-y-6">
              <div>
                <h4 className="font-bold text-lg text-white">M-Pesa Transaction Logs</h4>
                <p className="text-xs text-slate-500">Audit trail of all Lipa Na M-Pesa STK Push callbacks</p>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-slate-900 text-xs font-semibold uppercase tracking-wider text-slate-500">
                      <th className="py-4">Timestamp</th>
                      <th className="py-4">Checkout Request ID</th>
                      <th className="py-4">Receipt / Code</th>
                      <th className="py-4">Client Phone</th>
                      <th className="py-4">Amount</th>
                      <th className="py-4">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-900 text-sm">
                    {analytics?.recent_payments.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="py-8 text-center text-slate-500 font-semibold">
                          No transactions found.
                        </td>
                      </tr>
                    ) : (
                      analytics?.recent_payments.map((payment) => (
                        <tr key={payment.id} className="hover:bg-slate-900/10">
                          <td className="py-4 text-xs text-slate-400">
                            {new Date(payment.created_at).toLocaleString()}
                          </td>
                          <td className="py-4 font-mono text-xs text-slate-400">
                            {payment.checkout_request_id}
                          </td>
                          <td className="py-4 font-mono text-sm text-white">
                            {payment.transaction_id || '-'}
                          </td>
                          <td className="py-4 font-semibold text-slate-200">
                            {payment.phone_number}
                          </td>
                          <td className="py-4 font-bold text-orange-500">
                            KES {payment.amount_kes}
                          </td>
                          <td className="py-4">
                            <span className={`px-2 py-1 rounded-lg text-xs font-bold uppercase ${
                              payment.status === 'completed'
                                ? 'bg-green-500/10 text-green-500 border border-green-500/20'
                                : payment.status === 'pending'
                                ? 'bg-yellow-500/10 text-yellow-500 border border-yellow-500/20 animate-pulse'
                                : 'bg-red-500/10 text-red-500 border border-red-500/20'
                            }`}>
                              {payment.status}
                            </span>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {activeTab === 'plans' && (
            <div className="space-y-6">
              <div className="flex justify-between items-center">
                <div>
                  <h4 className="font-bold text-lg text-white">Billing Packages & Hotspot Plans</h4>
                  <p className="text-xs text-slate-500">Configure data and time packages displayed to clients on the portal landing screen</p>
                </div>
                <button
                  onClick={() => {
                    setEditingPlan(null);
                    setPlanForm({
                      name: '',
                      duration_minutes: 60,
                      price_kes: 20,
                      bandwidth_limit_down: 2048,
                      bandwidth_limit_up: 1024
                    });
                    setIsPlanModalOpen(true);
                  }}
                  className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-xl text-xs font-bold transition-all shadow-md shadow-orange-500/15"
                >
                  + Add Package
                </button>
              </div>

              <div className="bg-slate-900/20 border border-slate-900 rounded-3xl p-6">
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-b border-slate-900 text-xs font-semibold uppercase tracking-wider text-slate-500">
                        <th className="py-4">Package Name</th>
                        <th className="py-4">Access Time (Duration)</th>
                        <th className="py-4">Billing Rate (KES)</th>
                        <th className="py-4">Download Speed</th>
                        <th className="py-4">Upload Speed</th>
                        <th className="py-4 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-900 text-sm">
                      {plans.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="py-8 text-center text-slate-500 font-semibold">
                            No billing plans configured. Click "+ Add Package" to create one.
                          </td>
                        </tr>
                      ) : (
                        plans.map((p) => (
                          <tr key={p.id} className="hover:bg-slate-900/10">
                            <td className="py-4 font-bold text-white flex items-center gap-3">
                              <div className="p-2 bg-orange-500/10 border border-orange-500/20 text-orange-500 rounded-xl">
                                <Layers className="w-4 h-4" />
                              </div>
                              <span>{p.name}</span>
                            </td>
                            <td className="py-4 font-semibold text-slate-300">
                              {p.duration_minutes >= 1440 
                                ? `${Math.round(p.duration_minutes / 1440)} Day(s)` 
                                : p.duration_minutes >= 60 
                                ? `${Math.round(p.duration_minutes / 60)} Hour(s)` 
                                : `${p.duration_minutes} Mins`}
                              <span className="block text-[10px] text-slate-500 font-normal mt-0.5">({p.duration_minutes} total minutes)</span>
                            </td>
                            <td className="py-4 font-extrabold text-orange-500">
                              KES {p.price_kes}
                            </td>
                            <td className="py-4 font-semibold text-slate-300">
                              {p.bandwidth_limit_down > 0 ? `${p.bandwidth_limit_down / 1024} Mbps` : 'Unlimited'}
                            </td>
                            <td className="py-4 font-semibold text-slate-300">
                              {p.bandwidth_limit_up > 0 ? `${p.bandwidth_limit_up / 1024} Mbps` : 'Unlimited'}
                            </td>
                            <td className="py-4 text-right space-x-2">
                              <button
                                onClick={() => {
                                  setEditingPlan(p);
                                  setPlanForm({
                                    name: p.name,
                                    duration_minutes: p.duration_minutes,
                                    price_kes: p.price_kes,
                                    bandwidth_limit_down: p.bandwidth_limit_down,
                                    bandwidth_limit_up: p.bandwidth_limit_up
                                  });
                                  setIsPlanModalOpen(true);
                                }}
                                className="px-3 py-1.5 bg-slate-900 border border-slate-800 text-slate-300 hover:text-white rounded-lg text-xs font-bold transition-all"
                              >
                                Edit
                              </button>
                              <button
                                onClick={() => handleDeletePlan(p.id)}
                                className="px-3 py-1.5 bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white rounded-lg text-xs font-bold transition-all border border-red-500/20"
                              >
                                Delete
                              </button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Add/Edit Plan Modal */}
              {isPlanModalOpen && (
                <div className="fixed inset-0 bg-slate-950/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                  <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-md w-full space-y-6 shadow-2xl relative">
                    <div>
                      <h3 className="text-lg font-bold text-white">{editingPlan ? 'Edit Hotspot Package' : 'Create Hotspot Package'}</h3>
                      <p className="text-xs text-slate-500 mt-1">Configure pricing, duration, and speed limits</p>
                    </div>

                    <form onSubmit={handleSavePlan} className="space-y-4">
                      <div>
                        <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider mb-2">Package Name</label>
                        <input
                          type="text"
                          required
                          value={planForm.name}
                          onChange={(e) => setPlanForm({ ...planForm, name: e.target.value })}
                          className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                          placeholder="e.g. 1 Hour Plan"
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider mb-2">Duration (Mins)</label>
                          <input
                            type="number"
                            required
                            min="1"
                            value={planForm.duration_minutes}
                            onChange={(e) => setPlanForm({ ...planForm, duration_minutes: Number(e.target.value) })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                          />
                        </div>
                        <div>
                          <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider mb-2">Price (KES)</label>
                          <input
                            type="number"
                            required
                            min="0"
                            value={planForm.price_kes}
                            onChange={(e) => setPlanForm({ ...planForm, price_kes: Number(e.target.value) })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                          />
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider mb-2">Download Limit (Kbps)</label>
                          <input
                            type="number"
                            required
                            min="0"
                            value={planForm.bandwidth_limit_down}
                            onChange={(e) => setPlanForm({ ...planForm, bandwidth_limit_down: Number(e.target.value) })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                            placeholder="e.g. 2048 for 2 Mbps"
                          />
                          <span className="text-[10px] text-slate-500 mt-1 block">{(planForm.bandwidth_limit_down / 1024).toFixed(1)} Mbps</span>
                        </div>
                        <div>
                          <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider mb-2">Upload Limit (Kbps)</label>
                          <input
                            type="number"
                            required
                            min="0"
                            value={planForm.bandwidth_limit_up}
                            onChange={(e) => setPlanForm({ ...planForm, bandwidth_limit_up: Number(e.target.value) })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-500/10 transition-all font-semibold"
                            placeholder="e.g. 1024 for 1 Mbps"
                          />
                          <span className="text-[10px] text-slate-500 mt-1 block">{(planForm.bandwidth_limit_up / 1024).toFixed(1)} Mbps</span>
                        </div>
                      </div>

                      <div className="pt-4 flex justify-end gap-3 border-t border-slate-800">
                        <button
                          type="button"
                          onClick={() => setIsPlanModalOpen(false)}
                          className="px-4 py-2.5 bg-slate-950 border border-slate-800 text-slate-400 hover:text-white rounded-xl text-xs font-bold transition-all"
                        >
                          Cancel
                        </button>
                        <button
                          type="submit"
                          className="px-5 py-2.5 bg-orange-500 hover:bg-orange-600 text-white rounded-xl text-xs font-bold transition-all"
                        >
                          {editingPlan ? 'Save Changes' : 'Create Package'}
                        </button>
                      </div>
                    </form>
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'settings' && (
            <div className="bg-slate-900/20 border border-slate-900 rounded-3xl p-6 space-y-6">
              <div>
                <h4 className="font-bold text-lg text-white">Network & Payment Settings</h4>
                <p className="text-xs text-slate-500">System parameters and M-Pesa API credentials</p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                {/* Network config card */}
                <div className="bg-slate-900/40 border border-slate-900 p-6 rounded-2xl space-y-4">
                  <div className="flex items-center gap-3">
                    <Database className="w-5 h-5 text-indigo-500" />
                    <h5 className="font-bold text-white text-sm uppercase tracking-wide">Network Node Parameters</h5>
                  </div>
                  <div className="space-y-3 text-xs text-slate-300">
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>Gateway IP Address</span>
                      <span className="font-mono font-bold text-white">10.0.0.1</span>
                    </div>
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>Subnet Mask</span>
                      <span className="font-mono font-bold text-white">255.255.255.0</span>
                    </div>
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>DHCP Lease Range</span>
                      <span className="font-mono font-bold text-white">10.0.0.50 - 10.0.0.250</span>
                    </div>
                    <div className="flex justify-between py-2">
                      <span>Firewall Backend</span>
                      <span className="text-green-400 font-semibold uppercase">nftables (Simulation Mode)</span>
                    </div>
                  </div>
                </div>

                {/* M-Pesa credentials card */}
                <div className="bg-slate-900/40 border border-slate-900 p-6 rounded-2xl space-y-4">
                  <div className="flex items-center gap-3">
                    <DollarSign className="w-5 h-5 text-orange-500" />
                    <h5 className="font-bold text-white text-sm uppercase tracking-wide">Safaricom Daraja API</h5>
                  </div>
                  <div className="space-y-3 text-xs text-slate-300">
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>Environment</span>
                      <span className="font-bold text-yellow-500">Daraja Sandbox</span>
                    </div>
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>Business Shortcode</span>
                      <span className="font-mono font-bold text-white">174379</span>
                    </div>
                    <div className="flex justify-between py-2 border-b border-slate-900">
                      <span>Passkey</span>
                      <span className="font-mono font-bold text-white">bfb279f9aa9bdbcf...</span>
                    </div>
                    <div className="flex justify-between py-2">
                      <span>Callback URL</span>
                      <span className="font-mono font-bold text-white text-[10px] break-all">http://localhost:8082/payments/callback</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="pt-4 border-t border-slate-900 flex justify-end">
                <button className="px-5 py-2.5 bg-orange-500 text-white rounded-xl text-xs font-bold hover:bg-orange-600 transition-all shadow-md shadow-orange-500/10">
                  Save Configurations
                </button>
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
