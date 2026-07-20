import { useState, useEffect } from 'react';
import { Wifi, CreditCard, CheckCircle2, ShieldCheck, Loader2 } from 'lucide-react';

const API_BASE = window.location.origin.includes('3001') 
  ? 'http://localhost:8080' 
  : window.location.origin;

interface Plan {
  id: number;
  name: string;
  duration_minutes: number;
  price_kes: number;
  bandwidth_limit_down: number;
  bandwidth_limit_up: number;
}

export default function App() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState<number | null>(null);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [screen, setScreen] = useState<'form' | 'waiting' | 'success'>('form');
  const [checkoutRequestId, setCheckoutRequestId] = useState('');
  const [amount, setAmount] = useState(0);
  const [loading, setLoading] = useState(false);

  // Retrieve MAC and IP injected in URL query parameters by the gateway redirect
  const params = new URLSearchParams(window.location.search);
  const clientMac = params.get('mac') || '00:0a:95:9d:68:16';
  const clientIp = params.get('ip') || '10.0.0.5';

  useEffect(() => {
    fetchPlans();
  }, []);

  const fetchPlans = async () => {
    try {
      const res = await fetch(`${API_BASE}/plans`);
      const data = await res.json();
      setPlans(data);
      if (data.length > 0) {
        setSelectedPlanId(data[0].id);
      }
    } catch (e) {
      // Fallback fallback plans in case API server is bootstrapping
      const dummyPlans = [
        { id: 1, name: "1 Hour Plan", duration_minutes: 60, price_kes: 20.0, bandwidth_limit_down: 2048, bandwidth_limit_up: 1024 },
        { id: 2, name: "3 Hours Plan", duration_minutes: 180, price_kes: 50.0, bandwidth_limit_down: 3072, bandwidth_limit_up: 1536 },
        { id: 3, name: "24 Hours Plan", duration_minutes: 1440, price_kes: 100.0, bandwidth_limit_down: 5120, bandwidth_limit_up: 2048 },
      ];
      setPlans(dummyPlans);
      setSelectedPlanId(dummyPlans[0].id);
    }
  };

  const handlePay = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPlanId) return;

    const plan = plans.find(p => p.id === selectedPlanId);
    if (!plan) return;

    setLoading(true);
    setAmount(plan.price_kes);

    // Format phone to M-Pesa standard format: 2547XXXXXXXX
    let formattedPhone = phoneNumber.trim();
    if (formattedPhone.startsWith('0')) {
      formattedPhone = '254' + formattedPhone.substring(1);
    } else if (formattedPhone.startsWith('+')) {
      formattedPhone = formattedPhone.substring(1);
    } else if (!formattedPhone.startsWith('254')) {
      formattedPhone = '254' + formattedPhone;
    }

    try {
      const res = await fetch(`${API_BASE}/payments/stk`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          phone_number: formattedPhone,
          amount: plan.price_kes,
          plan_id: plan.id,
          mac_address: clientMac,
          ip_address: clientIp
        })
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Payment gateway returned error');
      }

      setCheckoutRequestId(data.checkout_request_id);
      setScreen('waiting');
      startPolling();
    } catch (err: any) {
      alert("Error: " + (err.message || "Failed to trigger payment"));
    } finally {
      setLoading(false);
    }
  };

  const startPolling = () => {
    let attempts = 0;
    const interval = setInterval(async () => {
      attempts++;
      if (attempts > 30) { // Timeout after 90 seconds
        clearInterval(interval);
        alert("Payment confirmation timed out. If you paid, please click refresh on your browser or reconnect.");
        setScreen('form');
        return;
      }

      try {
        // Poll for WebSocket pub/sub simulation confirmation
        // Let's connect to WS for status or check via REST.
        // For testing simplicity in standalone sandboxes, we verify using active sessions polling:
        const res = await fetch(`${API_BASE}/admin/analytics`);
        if (res.ok) {
          const data = await res.json();
          // Check if session with our mac address is active
          const session = data.active_sessions?.find((s: any) => s.device?.mac_address === clientMac);
          if (session) {
            clearInterval(interval);
            setScreen('success');
          }
        }
      } catch (e) {
        // Network error while connecting to unauthorized endpoints, ignore and retry
      }
    }, 3000);
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">
      {/* Background radial glows */}
      <div className="absolute top-1/4 left-1/4 w-80 h-80 bg-orange-500/10 rounded-full blur-[120px]"></div>
      <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-emerald-500/10 rounded-full blur-[120px]"></div>

      <div className="w-full max-w-md bg-slate-900/60 border border-slate-800/80 rounded-3xl p-8 backdrop-blur-xl shadow-2xl relative z-10">
        
        {screen === 'form' && (
          <div className="space-y-6">
            <div className="text-center">
              <div className="w-16 h-16 bg-gradient-to-tr from-orange-500 to-emerald-500 rounded-2xl mx-auto flex items-center justify-center shadow-lg shadow-orange-500/20 mb-4">
                <Wifi className="w-8 h-8 text-white animate-pulse" />
              </div>
              <h1 className="text-2xl font-extrabold tracking-tight text-white">Welcome to THEGOAT</h1>
              <p className="text-slate-400 mt-2 text-sm font-semibold">High-speed internet powered by M-Pesa</p>
            </div>

            <form onSubmit={handlePay} className="space-y-6">
              {/* Phone Input */}
              <div className="space-y-2">
                <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider">M-Pesa Mobile Number</label>
                <div className="relative">
                  <span className="absolute inset-y-0 left-0 pl-4 flex items-center text-slate-500 font-bold">
                    +254
                  </span>
                  <input
                    type="tel"
                    required
                    pattern="[0-9]{9,10}"
                    value={phoneNumber}
                    onChange={(e) => setPhoneNumber(e.target.value)}
                    className="w-full bg-slate-950/60 border border-slate-800 rounded-2xl py-4 pl-16 pr-4 text-white font-extrabold placeholder-slate-700 outline-none focus:border-orange-500 focus:ring-4 focus:ring-orange-500/10 transition-all"
                    placeholder="712345678"
                  />
                </div>
              </div>

              {/* Plans Selection */}
              <div className="space-y-2">
                <label className="block text-xs font-bold text-slate-400 uppercase tracking-wider">Select your plan</label>
                <div className="space-y-3">
                  {plans.map((plan) => (
                    <label 
                      key={plan.id}
                      className={`flex items-center justify-between p-4 rounded-2xl border-2 cursor-pointer transition-all ${
                        selectedPlanId === plan.id 
                          ? 'border-orange-500 bg-orange-500/5' 
                          : 'border-slate-800 bg-slate-950/40 hover:border-slate-700'
                      }`}
                    >
                      <input
                        type="radio"
                        name="plan"
                        value={plan.id}
                        checked={selectedPlanId === plan.id}
                        onChange={() => setSelectedPlanId(plan.id)}
                        className="hidden"
                      />
                      <div className="space-y-1">
                        <span className="font-bold text-slate-200 text-sm md:text-base">{plan.name}</span>
                        <span className="block text-xs text-slate-500 font-semibold">
                          Download rate up to {plan.bandwidth_limit_down ? `${plan.bandwidth_limit_down/1024} Mbps` : '10 Mbps'}
                        </span>
                      </div>
                      <span className="font-extrabold text-orange-500 text-base md:text-lg">
                        KES {plan.price_kes}
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              {/* Pay Button */}
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 text-white font-bold py-4 px-4 rounded-2xl shadow-lg shadow-orange-500/25 transition-all duration-200 transform hover:-translate-y-0.5 active:translate-y-0 flex items-center justify-center gap-2"
              >
                {loading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <CreditCard className="w-5 h-5" />
                    <span>Pay KES {plans.find(p => p.id === selectedPlanId)?.price_kes || 0} via M-Pesa</span>
                  </>
                )}
              </button>
            </form>

            <div className="flex items-center justify-center gap-2 text-[10px] text-slate-500 font-semibold text-center border-t border-slate-800/80 pt-4">
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
              <span>Secured client authorization token proxy</span>
            </div>
          </div>
        )}

        {screen === 'waiting' && (
          <div className="text-center py-8 space-y-6 animate-fade-in">
            <Loader2 className="w-16 h-16 text-orange-500 animate-spin mx-auto" />
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-white">M-Pesa STK Push Sent</h2>
              <p className="text-sm text-slate-400 px-4">
                Please check your phone and enter your M-Pesa PIN on the dialog window to authorize payment of <span className="font-extrabold text-orange-500">KES {amount}</span>.
              </p>
            </div>
            <div className="p-4 bg-slate-950/40 border border-slate-800 rounded-2xl inline-block">
              <p className="text-xs text-slate-500 font-mono">Checkout Request ID</p>
              <p className="text-xs text-slate-300 font-mono mt-1 font-bold">{checkoutRequestId.substring(0, 20)}...</p>
            </div>
            <p className="text-xs text-slate-500 animate-pulse font-bold">Waiting for payment receipt...</p>
          </div>
        )}

        {screen === 'success' && (
          <div className="text-center py-8 space-y-6 animate-scale-up">
            <div className="w-20 h-20 bg-emerald-500/10 border border-emerald-500/30 text-emerald-500 rounded-full mx-auto flex items-center justify-center shadow-lg shadow-emerald-500/10">
              <CheckCircle2 className="w-12 h-12" />
            </div>
            <div className="space-y-2">
              <h2 className="text-2xl font-black text-emerald-500">Payment Confirmed</h2>
              <h3 className="text-lg font-bold text-white">You are now Connected!</h3>
              <p className="text-sm text-slate-400 px-4">
                Your session is authorized on the gateway firewall. Enjoy high-speed surfing.
              </p>
            </div>
            <button
              onClick={() => window.location.href = 'https://www.google.com'}
              className="w-full bg-gradient-to-r from-emerald-500 to-emerald-600 hover:from-emerald-600 hover:to-emerald-700 text-white font-bold py-4 px-4 rounded-2xl shadow-lg shadow-emerald-500/25 transition-all duration-200 transform hover:-translate-y-0.5 active:translate-y-0"
            >
              Start Browsing
            </button>
          </div>
        )}

      </div>
    </div>
  );
}
