package billing

// OperationType identifies a billable API operation.
type OperationType string

const (
	OpDomainCheck    OperationType = "domain_check"    // free
	OpDomainRegister OperationType = "domain_register" // cost-based + markup
	OpDomainList     OperationType = "domain_list"     // fixed fee
	OpDomainInfo     OperationType = "domain_info"     // fixed fee
	OpDNSList        OperationType = "dns_list"         // fixed fee
	OpDNSAdd         OperationType = "dns_add"          // fixed fee
	OpDNSDelete      OperationType = "dns_delete"       // fixed fee
	OpDomainRenew    OperationType = "domain_renew"    // fixed fee
	OpNSUpdate       OperationType = "ns_update"       // fixed fee
	OpDNSUpdate        OperationType = "dns_update"        // fixed fee
	OpDomainCheckPaid  OperationType = "domain_check_paid" // paid check over free quota
)

const (
	MarkupPercent    = 20
	FixedFeeCents    = 10 // $0.10 for DNS mutations
	ListFeeCents     = 5  // $0.05 for list/info operations
	MinTopUpCents    = 500 // $5.00 minimum top-up
	CheckFeeCents    = 1   // $0.01 per check over free quota
)

// CalculateCostCents returns the cost in cents for the given operation.
// For OpDomainRegister, baseCostCents is the registrar's price.
// For other operations, baseCostCents is ignored.
func CalculateCostCents(op OperationType, baseCostCents int64) int64 {
	switch op {
	case OpDomainCheck:
		return 0
	case OpDomainCheckPaid:
		return CheckFeeCents
	case OpDomainRegister:
		return baseCostCents + (baseCostCents * MarkupPercent / 100)
	case OpDomainList, OpDomainInfo, OpDNSList:
		return ListFeeCents
	case OpDNSAdd, OpDNSDelete, OpDomainRenew, OpNSUpdate, OpDNSUpdate:
		return FixedFeeCents
	default:
		return 0
	}
}
