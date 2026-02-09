package spaceship

// SpaceshipPricing contains known Spaceship domain prices [registration, renewal] in USD.
var SpaceshipPricing = map[string][2]float64{
	"com":      {9.08, 10.18},
	"net":      {11.40, 11.40},
	"org":      {6.68, 10.00},
	"io":       {31.98, 51.75},
	"dev":      {10.55, 12.62},
	"sh":       {31.05, 46.58},
	"app":      {14.69, 14.69},
	"xyz":      {2.06, 12.72},
	"info":     {3.31, 21.94},
	"me":       {15.53, 15.53},
	"co":       {22.98, 22.98},
	"cc":       {8.98, 8.98},
	"ai":       {55.98, 55.98},
	"gg":       {39.98, 39.98},
	"cloud":    {15.98, 15.98},
	"run":      {15.98, 15.98},
	"tools":    {22.98, 22.98},
	"codes":    {39.98, 39.98},
	"software": {22.98, 22.98},
	"biz":      {13.98, 13.98},
	"pro":      {13.98, 13.98},
	"us":       {7.98, 7.98},
	"uk":       {7.98, 7.98},
	"de":       {7.98, 7.98},
	"eu":       {7.98, 7.98},
	"ca":       {11.98, 11.98},
	"in":       {7.98, 7.98},
	"tech":     {39.98, 39.98},
	"site":     {2.98, 22.98},
	"online":   {2.98, 29.98},
	"store":    {2.98, 39.98},
}

// GetStaticPrice returns Spaceship's known price for a TLD.
func GetStaticPrice(tld string) (reg, renew float64) {
	p, found := SpaceshipPricing[tld]
	if !found {
		return 0, 0
	}
	return p[0], p[1]
}
