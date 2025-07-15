import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { domains as domainsApi } from '../lib/api'

export default function Domains() {
  const queryClient = useQueryClient()
  const [showAddModal, setShowAddModal] = useState(false)
  const [newDomain, setNewDomain] = useState({ name: '', registrar: 'value-domain' })

  const { data, isLoading } = useQuery({
    queryKey: ['domains'],
    queryFn: () => domainsApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: domainsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domains'] })
      setShowAddModal(false)
      setNewDomain({ name: '', registrar: 'value-domain' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: domainsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domains'] })
    },
  })

  const handleAddDomain = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(newDomain)
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Domains</h1>
          <p className="mt-2 text-slate-600 text-lg">
            Manage your domains across multiple registrars
          </p>
        </div>
        <div className="mt-4 sm:mt-0">
          <button
            onClick={() => setShowAddModal(true)}
            className="inline-flex items-center px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-medium rounded-xl hover:from-blue-600 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-all transform hover:scale-105 active:scale-95"
          >
            <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add Domain
          </button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex justify-center items-center py-20">
          <div className="relative">
            <div className="w-16 h-16 border-4 border-blue-200 rounded-full animate-spin"></div>
            <div className="absolute top-0 left-0 w-16 h-16 border-4 border-transparent border-t-blue-500 rounded-full animate-spin"></div>
          </div>
        </div>
      ) : (
        <div className="bg-white/60 backdrop-blur-sm border border-slate-200/50 rounded-2xl overflow-hidden">
          {data?.domains && data.domains.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200/50">
                <thead className="bg-slate-50/50">
                  <tr>
                    <th scope="col" className="px-6 py-4 text-left text-sm font-semibold text-slate-900">
                      Domain
                    </th>
                    <th scope="col" className="px-6 py-4 text-left text-sm font-semibold text-slate-900">
                      Registrar
                    </th>
                    <th scope="col" className="px-6 py-4 text-left text-sm font-semibold text-slate-900">
                      Status
                    </th>
                    <th scope="col" className="px-6 py-4 text-left text-sm font-semibold text-slate-900">
                      Expires
                    </th>
                    <th scope="col" className="relative px-6 py-4">
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/50 bg-white/30">
                  {data.domains.map((domain: any) => (
                    <tr key={domain.id} className="hover:bg-slate-50/50 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="w-3 h-3 bg-blue-400 rounded-full mr-3"></div>
                          <span className="text-sm font-medium text-slate-900">{domain.name}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                        <span className="px-3 py-1 bg-slate-100 text-slate-700 rounded-full text-xs font-medium">
                          {domain.registrar}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="inline-flex px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          {domain.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                        {domain.expires_at ? new Date(domain.expires_at).toLocaleDateString() : 'N/A'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button
                          onClick={() => deleteMutation.mutate(domain.id)}
                          className="text-red-500 hover:text-red-700 transition-colors p-2 rounded-lg hover:bg-red-50"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="text-center py-16">
              <div className="w-20 h-20 bg-slate-100 rounded-3xl flex items-center justify-center mx-auto mb-6">
                <svg className="w-10 h-10 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9-9a9 9 0 00-9 9m0 0a9 9 0 009 9" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-slate-900 mb-2">No domains yet</h3>
              <p className="text-slate-500 mb-6">Add your first domain to get started with domain management.</p>
              <button
                onClick={() => setShowAddModal(true)}
                className="inline-flex items-center px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
              >
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
                Add Domain
              </button>
            </div>
          )}
        </div>
      )}

      {/* Add Domain Modal */}
      {showAddModal && (
        <div className="fixed z-50 inset-0 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            <div className="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity" onClick={() => setShowAddModal(false)}></div>

            <div className="inline-block align-bottom bg-white/95 backdrop-blur-xl rounded-3xl text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full border border-white/20">
              <form onSubmit={handleAddDomain}>
                <div className="px-8 pt-8 pb-6">
                  <div className="flex items-center mb-6">
                    <div className="w-12 h-12 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center mr-4">
                      <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9-9a9 9 0 00-9 9m0 0a9 9 0 009 9" />
                      </svg>
                    </div>
                    <h3 className="text-2xl font-bold text-slate-900">Add Domain</h3>
                  </div>
                  
                  <div className="space-y-6">
                    <div>
                      <label htmlFor="domain" className="block text-sm font-medium text-slate-700 mb-2">
                        Domain Name
                      </label>
                      <input
                        type="text"
                        id="domain"
                        value={newDomain.name}
                        onChange={(e) => setNewDomain({ ...newDomain, name: e.target.value })}
                        className="w-full px-4 py-3 bg-white/50 border border-slate-200 rounded-xl text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                        placeholder="example.com"
                        required
                      />
                    </div>

                    <div>
                      <label htmlFor="registrar" className="block text-sm font-medium text-slate-700 mb-2">
                        Registrar
                      </label>
                      <select
                        id="registrar"
                        value={newDomain.registrar}
                        onChange={(e) => setNewDomain({ ...newDomain, registrar: e.target.value })}
                        className="w-full px-4 py-3 bg-white/50 border border-slate-200 rounded-xl text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                      >
                        <option value="value-domain">VALUE-DOMAIN</option>
                        <option value="route53">Route 53</option>
                        <option value="porkbun">Porkbun</option>
                      </select>
                    </div>
                  </div>
                </div>

                <div className="bg-slate-50/50 px-8 py-6 flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-3 space-y-3 space-y-reverse sm:space-y-0">
                  <button
                    type="button"
                    onClick={() => setShowAddModal(false)}
                    className="inline-flex justify-center px-6 py-3 border border-slate-200 bg-white text-slate-700 rounded-xl hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-slate-500 transition-all"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={createMutation.isPending}
                    className="inline-flex justify-center px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-xl hover:from-blue-600 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-all transform hover:scale-105 active:scale-95"
                  >
                    {createMutation.isPending ? (
                      <div className="flex items-center">
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2"></div>
                        Adding...
                      </div>
                    ) : (
                      'Add Domain'
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}