package change

// Versioned, sourced GCP price list for the deterministic cost calculator
// (calculator.go). Every PriceRecord below was read by hand from the cited
// official GCP pricing page on AsOf — none is filled from model memory. A
// price this build could not verify stays Known: false, and the calculator
// must then report the affected cost line as UNKNOWN instead of guessing.
//
// Bump PriceTableVersion whenever any PriceRecord's UnitPrice, Unit,
// Currency, Region or SourceURL changes, so every CostEstimate stays
// traceable to the exact snapshot that produced it.
const PriceTableVersion = "gcp-pricing-2026-09-27"

// PriceRecord is one line of the versioned price table.
type PriceRecord struct {
	SKU         string
	Description string
	// UnitPrice is meaningless when Known is false; the calculator must not
	// read it in that case.
	UnitPrice float64
	Unit      string // "GiB-month" (storage) or "GiB" (egress, per transferred gibibyte)
	Currency  string
	Region    string
	// OriginRegion is the short GCP region code this price was actually
	// queried for (e.g. "us-central1"). Region above additionally spells out
	// the destination side for egress and is meant for the full traceability
	// trail; OriginRegion is the short label the workspace surfaces next to
	// an estimate so a reader can compare it against the deployment region
	// (see DR-014 cost_assumptions: pricing is us-central1, the deploy
	// default is asia-east1).
	OriginRegion string
	SourceURL    string
	AsOf         string // ISO date the price was read from SourceURL
	Known        bool
}

// PriceTable is the static price list this build ships with. It only covers
// the two SKUs the P0-C demo candidate (managed_storage_with_signed_urls)
// actually uses: Cloud Storage Standard storage and general network egress.
// Extending coverage to other SKUs or options is future work, not something
// the calculator infers.
var PriceTable = map[string]PriceRecord{
	// Read 2026-09-27 from https://cloud.google.com/storage/pricing
	// ("Data Storage pricing" table, region selector set to
	// "Iowa (us-central1)"): the Standard storage row lists
	// $0.000027397 per gibibyte-hour, per account/month, which is the
	// same $0.02/GiB-month figure GCP has long quoted for Standard storage
	// in us-central1 (0.000027397 * 730 average hours/month = 0.02).
	// The page shows no separate "last updated" date; AsOf is the date this
	// price was read live.
	//
	// RESERVED, NOT USED IN THIS SLICE: EstimateOptionCost (calculator.go)
	// does not read this entry. Pricing "stored bytes over retention" needs
	// a retention-period business constraint that does not exist yet (see
	// DR-014's rejected alternative "price_all_three_named_cost_drivers");
	// that line item stays an explicit UNKNOWN instead. This record is kept
	// here, priced and sourced, so wiring it up later is a calculator change
	// only, not another pricing lookup.
	"gcs-standard-storage": {
		SKU:          "gcs-standard-storage",
		Description:  "Cloud Storage Standard storage class, regional (us-central1)",
		UnitPrice:    0.02,
		Unit:         "GiB-month",
		Currency:     "USD",
		Region:       "us-central1",
		OriginRegion: "us-central1",
		SourceURL:    "https://cloud.google.com/storage/pricing#storage-pricing",
		AsOf:         "2026-09-27",
		Known:        true,
	},
	// Read 2026-09-27 from https://cloud.google.com/storage/pricing
	// ("Network" > "General network usage" table): "Data transfer to
	// Worldwide Destinations (excluding Asia & Australia)", first tier
	// (0 GiB to 10 TiB per month): $0.12 / GiB.
	"gcs-network-egress-worldwide-tier1": {
		SKU:          "gcs-network-egress-worldwide-tier1",
		Description:  "Cloud Storage network egress to worldwide destinations (excluding Asia & Australia), 0-10 TiB/month tier",
		UnitPrice:    0.12,
		Unit:         "GiB",
		Currency:     "USD",
		Region:       "origin us-central1 / destination worldwide excl. Asia & Australia",
		OriginRegion: "us-central1",
		SourceURL:    "https://cloud.google.com/storage/pricing#network-egress",
		AsOf:         "2026-09-27",
		Known:        true,
	},
}
