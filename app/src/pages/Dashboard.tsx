import { useQuery } from '@tanstack/react-query'
import { billing, domains } from '../lib/api'
import { useAuthStore } from '../stores/auth'

export default function Dashboard() {
  const user = useAuthStore((state) => state.user)

  const { data: usage } = useQuery({
    queryKey: ['usage'],
    queryFn: billing.getUsage,
  })

  const { data: domainList } = useQuery({
    queryKey: ['domains'],
    queryFn: () => domains.list({ limit: 5 }),
  })

  return (
    <div className="space-y-8">
      {/* Welcome Header */}
      <div className="relative overflow-hidden bg-gradient-to-r from-blue-600 to-purple-600 rounded-3xl p-8">
        <div className="absolute inset-0 bg-black/20 backdrop-blur-sm"></div>
        <div className="relative">
          <h1 className="text-3xl font-bold text-white">
            Welcome back, {user?.name || user?.email?.split('@')[0]}
          </h1>
          <p className="mt-2 text-blue-100 text-lg">
            Manage your domains and DNS records from a single dashboard
          </p>
        </div>
        <div className="absolute -right-12 -bottom-12 w-32 h-32 bg-white/10 rounded-full blur-2xl"></div>
        <div className="absolute -right-6 -top-6 w-20 h-20 bg-white/20 rounded-full blur-xl"></div>
      </div>

      {/* Plan Overview */}
      <div className="bg-gradient-to-r from-slate-50 to-blue-50 border border-blue-100 rounded-2xl p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-xl font-bold text-slate-900">
              Current Plan: {user?.subscriptionPlan || 'Free'}
            </h3>
            <p className="text-slate-600 mt-1">
              {user?.subscriptionPlan === 'free' ? 'Perfect for getting started' : 'Professional domain management'}
            </p>
          </div>
          <div className="w-16 h-16 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center">
            <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
            </svg>
          </div>
        </div>
      </div>

      {/* Usage Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl p-6 hover:shadow-lg transition-all duration-200">
          <div className="flex items-center justify-between">
            <div>
              <dt className="text-sm font-medium text-slate-600">Domains</dt>
              <dd className="mt-2 text-3xl font-bold text-slate-900">
                {usage?.usage.domains.used || 0}
                <span className="text-lg text-slate-500 ml-1">/ {usage?.usage.domains.limit || 3}</span>
              </dd>
            </div>
            <div className="w-12 h-12 bg-gradient-to-r from-blue-500 to-blue-600 rounded-xl flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9-9a9 9 0 00-9 9m0 0a9 9 0 009 9" />
              </svg>
            </div>
          </div>
          <div className="mt-4 bg-slate-100 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-blue-500 to-blue-600 h-2 rounded-full transition-all duration-500"
              style={{ width: `${Math.min(((usage?.usage.domains.used || 0) / (usage?.usage.domains.limit || 3)) * 100, 100)}%` }}
            ></div>
          </div>
        </div>

        <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl p-6 hover:shadow-lg transition-all duration-200">
          <div className="flex items-center justify-between">
            <div>
              <dt className="text-sm font-medium text-slate-600">DNS Records</dt>
              <dd className="mt-2 text-3xl font-bold text-slate-900">
                {usage?.usage.dnsRecords.used || 0}
                <span className="text-lg text-slate-500 ml-1">/ {usage?.usage.dnsRecords.limit || 100}</span>
              </dd>
            </div>
            <div className="w-12 h-12 bg-gradient-to-r from-purple-500 to-purple-600 rounded-xl flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
          </div>
          <div className="mt-4 bg-slate-100 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-purple-500 to-purple-600 h-2 rounded-full transition-all duration-500"
              style={{ width: `${Math.min(((usage?.usage.dnsRecords.used || 0) / (usage?.usage.dnsRecords.limit || 100)) * 100, 100)}%` }}
            ></div>
          </div>
        </div>

        <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl p-6 hover:shadow-lg transition-all duration-200">
          <div className="flex items-center justify-between">
            <div>
              <dt className="text-sm font-medium text-slate-600">API Calls</dt>
              <dd className="mt-2 text-3xl font-bold text-slate-900">
                {usage?.usage.apiCalls.used || 0}
                <span className="text-lg text-slate-500 ml-1">/ {usage?.usage.apiCalls.limit || 1000}</span>
              </dd>
            </div>
            <div className="w-12 h-12 bg-gradient-to-r from-green-500 to-green-600 rounded-xl flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
          </div>
          <div className="mt-4 bg-slate-100 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-green-500 to-green-600 h-2 rounded-full transition-all duration-500"
              style={{ width: `${Math.min(((usage?.usage.apiCalls.used || 0) / (usage?.usage.apiCalls.limit || 1000)) * 100, 100)}%` }}
            ></div>
          </div>
        </div>

        <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl p-6 hover:shadow-lg transition-all duration-200">
          <div className="flex items-center justify-between">
            <div>
              <dt className="text-sm font-medium text-slate-600">API Keys</dt>
              <dd className="mt-2 text-3xl font-bold text-slate-900">
                {usage?.usage.apiKeys.used || 0}
                <span className="text-lg text-slate-500 ml-1">/ {usage?.usage.apiKeys.limit || 1}</span>
              </dd>
            </div>
            <div className="w-12 h-12 bg-gradient-to-r from-orange-500 to-orange-600 rounded-xl flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
              </svg>
            </div>
          </div>
          <div className="mt-4 bg-slate-100 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-orange-500 to-orange-600 h-2 rounded-full transition-all duration-500"
              style={{ width: `${Math.min(((usage?.usage.apiKeys.used || 0) / (usage?.usage.apiKeys.limit || 1)) * 100, 100)}%` }}
            ></div>
          </div>
        </div>
      </div>

      {/* Recent Domains */}
      <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200/50">
          <h3 className="text-lg font-semibold text-slate-900">Recent Domains</h3>
        </div>
        <div className="divide-y divide-slate-200/50">
          {domainList?.domains?.length > 0 ? (
            domainList.domains.map((domain: any) => (
              <div key={domain.id} className="px-6 py-4 hover:bg-slate-50/50 transition-colors">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <div className="w-3 h-3 bg-green-400 rounded-full"></div>
                    <p className="text-sm font-medium text-slate-900">
                      {domain.name}
                    </p>
                  </div>
                  <div className="flex items-center space-x-3">
                    <span className="px-3 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                      {domain.status}
                    </span>
                    <span className="text-sm text-slate-500">
                      {domain.registrar}
                    </span>
                  </div>
                </div>
              </div>
            ))
          ) : (
            <div className="px-6 py-8 text-center">
              <div className="w-16 h-16 bg-slate-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9-9a9 9 0 00-9 9m0 0a9 9 0 009 9" />
                </svg>
              </div>
              <p className="text-slate-500">No domains yet. Add your first domain to get started.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}