import { useQuery, useMutation } from '@tanstack/react-query'
import { billing } from '../lib/api'

export default function Billing() {
  const { data: plans } = useQuery({
    queryKey: ['plans'],
    queryFn: billing.getPlans,
  })

  const { data: subscription } = useQuery({
    queryKey: ['subscription'],
    queryFn: billing.getSubscription,
  })

  const checkoutMutation = useMutation({
    mutationFn: ({ planId, interval }: { planId: string; interval: 'month' | 'year' }) =>
      billing.createCheckout(planId, interval),
    onSuccess: (data) => {
      window.location.href = data.checkoutUrl
    },
  })

  const cancelMutation = useMutation({
    mutationFn: billing.cancelSubscription,
  })

  const currentPlan = subscription?.plan?.id || 'free'

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-slate-900">Billing & Plans</h1>
        <p className="mt-2 text-slate-600 text-lg">
          Manage your subscription and billing information
        </p>
      </div>

      {/* Current Plan */}
      {subscription && (
        <div className="bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-100 rounded-2xl p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xl font-bold text-slate-900 flex items-center">
                <div className="w-3 h-3 bg-green-400 rounded-full mr-3"></div>
                Current Plan: {subscription.plan.name}
              </h3>
              <div className="mt-2 text-slate-600">
                <p>
                  {subscription.plan.id === 'free' 
                    ? 'Perfect for getting started with domain management'
                    : 'Professional domain management with advanced features'
                  }
                </p>
                {subscription.expiresAt && (
                  <p className="mt-1 text-sm">
                    {subscription.status === 'canceling' 
                      ? `Your subscription will end on ${new Date(subscription.expiresAt).toLocaleDateString()}`
                      : `Next billing date: ${new Date(subscription.expiresAt).toLocaleDateString()}`}
                  </p>
                )}
              </div>
            </div>
            
            {subscription.plan.id !== 'free' && subscription.status !== 'canceling' && (
              <button
                onClick={() => cancelMutation.mutate()}
                className="px-4 py-2 border border-red-200 text-red-600 bg-white rounded-xl hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500 transition-all"
              >
                Cancel Subscription
              </button>
            )}
          </div>
        </div>
      )}

      {/* Pricing Plans */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {plans && Object.values(plans).map((plan: any, index: number) => (
          <div
            key={plan.id}
            className={`relative rounded-2xl border-2 transition-all duration-200 hover:shadow-xl ${
              currentPlan === plan.id 
                ? 'border-blue-500 bg-gradient-to-b from-blue-50 to-white shadow-lg' 
                : 'border-slate-200 bg-white/60 backdrop-blur-sm hover:border-blue-300'
            } ${index === 1 ? 'lg:scale-105 lg:shadow-xl' : ''}`}
          >
            {currentPlan === plan.id && (
              <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                <span className="inline-flex rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 px-4 py-1 text-sm font-semibold text-white shadow-lg">
                  Current Plan
                </span>
              </div>
            )}

            {index === 1 && currentPlan !== plan.id && (
              <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                <span className="inline-flex rounded-xl bg-gradient-to-r from-orange-500 to-pink-600 px-4 py-1 text-sm font-semibold text-white shadow-lg">
                  Most Popular
                </span>
              </div>
            )}
            
            <div className="p-8">
              <div className="text-center">
                <h3 className="text-2xl font-bold text-slate-900">{plan.name}</h3>
                <div className="mt-4">
                  <span className="text-5xl font-extrabold text-slate-900">
                    ${plan.price / 100}
                  </span>
                  {plan.interval && (
                    <span className="text-lg font-medium text-slate-500">/{plan.interval}</span>
                  )}
                </div>
                {plan.id === 'free' && (
                  <p className="mt-2 text-sm text-slate-500">Forever free</p>
                )}
              </div>
              
              <ul className="mt-8 space-y-4">
                {[
                  { 
                    icon: '🌐', 
                    text: `${plan.features.domains === -1 ? 'Unlimited' : plan.features.domains} domains`,
                    highlight: plan.features.domains === -1
                  },
                  { 
                    icon: '📝', 
                    text: `${plan.features.dnsRecords === -1 ? 'Unlimited' : plan.features.dnsRecords.toLocaleString()} DNS records`,
                    highlight: plan.features.dnsRecords === -1
                  },
                  { 
                    icon: '⚡', 
                    text: `${plan.features.apiCalls === -1 ? 'Unlimited' : plan.features.apiCalls.toLocaleString()} API calls/month`,
                    highlight: plan.features.apiCalls === -1
                  },
                  { 
                    icon: '🔗', 
                    text: `${plan.features.webhooks === -1 ? 'Unlimited' : plan.features.webhooks} webhooks`,
                    highlight: plan.features.webhooks > 0
                  },
                  { 
                    icon: '🔑', 
                    text: `${plan.features.apiKeys === -1 ? 'Unlimited' : plan.features.apiKeys} API keys`,
                    highlight: plan.features.apiKeys === -1 || plan.features.apiKeys > 1
                  },
                  { 
                    icon: '🛡️', 
                    text: `${plan.features.support} support`,
                    highlight: plan.features.support !== 'community'
                  }
                ].map((feature, i) => (
                  <li key={i} className="flex items-center">
                    <span className="text-lg mr-3">{feature.icon}</span>
                    <span className={`text-sm ${feature.highlight ? 'text-slate-900 font-medium' : 'text-slate-600'}`}>
                      {feature.text}
                    </span>
                  </li>
                ))}
              </ul>

              <div className="mt-8">
                {plan.id === 'free' ? (
                  <button
                    disabled
                    className="w-full rounded-xl border-2 border-slate-200 bg-slate-50 py-3 px-4 text-sm font-medium text-slate-400 cursor-not-allowed"
                  >
                    {currentPlan === 'free' ? 'Current Plan' : 'Free Plan'}
                  </button>
                ) : currentPlan === plan.id ? (
                  <button
                    disabled
                    className="w-full rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 py-3 px-4 text-sm font-medium text-white shadow-lg cursor-not-allowed opacity-75"
                  >
                    Current Plan
                  </button>
                ) : (
                  <button
                    onClick={() => checkoutMutation.mutate({ planId: plan.id, interval: 'month' })}
                    disabled={checkoutMutation.isPending}
                    className="w-full rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 py-3 px-4 text-sm font-medium text-white shadow-lg hover:from-blue-600 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-all transform hover:scale-105 active:scale-95"
                  >
                    {checkoutMutation.isPending ? (
                      <div className="flex items-center justify-center">
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2"></div>
                        Processing...
                      </div>
                    ) : (
                      'Upgrade Now'
                    )}
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* FAQ Section */}
      <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl p-8">
        <h3 className="text-xl font-bold text-slate-900 mb-6">Frequently Asked Questions</h3>
        <div className="space-y-4">
          <div>
            <h4 className="font-medium text-slate-900">Can I change plans anytime?</h4>
            <p className="text-sm text-slate-600 mt-1">Yes, you can upgrade or downgrade your plan at any time. Changes take effect immediately.</p>
          </div>
          <div>
            <h4 className="font-medium text-slate-900">What payment methods do you accept?</h4>
            <p className="text-sm text-slate-600 mt-1">We accept all major credit cards, PayPal, and bank transfers for enterprise plans.</p>
          </div>
          <div>
            <h4 className="font-medium text-slate-900">Is there a free trial?</h4>
            <p className="text-sm text-slate-600 mt-1">Our Free plan is permanent and includes all basic features. You can upgrade anytime.</p>
          </div>
        </div>
      </div>
    </div>
  )
}