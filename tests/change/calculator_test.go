package change_test

import (
	"reflect"
	"testing"

	"context-rail/internal/change"
)

// storageConstraints is a business_constraints map with a volume string the
// calculator's parser recognises.
func storageConstraints(volume string) map[string]string {
	return map[string]string{"data_classification": "internal", "expected_monthly_volume": volume}
}

func TestEstimateOptionCostIsDeterministic(t *testing.T) {
	constraints := storageConstraints("500 GB")
	first := change.EstimateOptionCost("managed_storage_with_signed_urls", constraints)
	second := change.EstimateOptionCost("managed_storage_with_signed_urls", constraints)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same input must produce the same estimate:\n%+v\n%+v", first, second)
	}
	if first.Status != change.CostEstimated {
		t.Fatalf("expected ESTIMATED, got %+v", first)
	}
}

func TestEstimateOptionCostUnknownOptionHasNoSKU(t *testing.T) {
	estimate := change.EstimateOptionCost("minimal_reversible_slice", storageConstraints("500 GB"))
	if estimate.Status != change.CostUnknown {
		t.Fatalf("expected UNKNOWN for an option with no priced SKU, got %+v", estimate)
	}
	if estimate.Reason == "" {
		t.Fatal("an UNKNOWN estimate must say why")
	}
	if estimate.PriceTableVersion != change.PriceTableVersion {
		t.Fatalf("expected price_table_version to be recorded even when UNKNOWN, got %q", estimate.PriceTableVersion)
	}
}

func TestEstimateOptionCostUnparseableVolumeIsUnknown(t *testing.T) {
	cases := []string{"500 reviews", "10000 uploads", "", "a lot", "GB 500", "500"}
	for _, volume := range cases {
		estimate := change.EstimateOptionCost("managed_storage_with_signed_urls", storageConstraints(volume))
		if estimate.Status != change.CostUnknown {
			t.Fatalf("volume %q must be UNKNOWN, got %+v", volume, estimate)
		}
		if estimate.MonthlyLow != 0 || estimate.MonthlyBase != 0 || estimate.MonthlyHigh != 0 {
			t.Fatalf("an UNKNOWN estimate must not carry invented amounts: %+v", estimate)
		}
		for _, item := range estimate.Breakdown {
			if item.Status == change.CostEstimated {
				t.Fatalf("no breakdown line may be ESTIMATED when the volume did not parse: %+v", item)
			}
		}
	}
}

func TestEstimateOptionCostMissingBusinessConstraintIsUnknown(t *testing.T) {
	estimate := change.EstimateOptionCost("managed_storage_with_signed_urls", map[string]string{"data_classification": "internal"})
	if estimate.Status != change.CostUnknown {
		t.Fatalf("expected UNKNOWN with no expected_monthly_volume declared, got %+v", estimate)
	}
}

func TestEstimateOptionCostScenarioArithmetic(t *testing.T) {
	estimate := change.EstimateOptionCost("managed_storage_with_signed_urls", storageConstraints("1000 GB"))
	if estimate.Status != change.CostEstimated {
		t.Fatalf("expected ESTIMATED, got %+v", estimate)
	}
	price := change.PriceTable["gcs-network-egress-worldwide-tier1"]
	if !price.Known {
		t.Fatal("test assumes gcs-network-egress-worldwide-tier1 is a known price; update the test if the price table changes")
	}
	wantBase := 1000 * change.BaseScenarioMultiplier * price.UnitPrice
	wantLow := 1000 * change.LowScenarioMultiplier * price.UnitPrice
	wantHigh := 1000 * change.HighScenarioMultiplier * price.UnitPrice
	if estimate.MonthlyBase != wantBase || estimate.MonthlyLow != wantLow || estimate.MonthlyHigh != wantHigh {
		t.Fatalf("expected low/base/high %v/%v/%v, got %v/%v/%v", wantLow, wantBase, wantHigh, estimate.MonthlyLow, estimate.MonthlyBase, estimate.MonthlyHigh)
	}
	if estimate.MonthlyLow >= estimate.MonthlyBase || estimate.MonthlyBase >= estimate.MonthlyHigh {
		t.Fatalf("expected low < base < high, got %+v", estimate)
	}
	if estimate.ParsedVolumeGiB == nil || *estimate.ParsedVolumeGiB != 1000 {
		t.Fatalf("expected parsed volume 1000 GiB, got %+v", estimate.ParsedVolumeGiB)
	}
}

func TestEstimateOptionCostTraceability(t *testing.T) {
	estimate := change.EstimateOptionCost("managed_storage_with_signed_urls", storageConstraints("2 TB"))
	if estimate.Status != change.CostEstimated {
		t.Fatalf("expected ESTIMATED, got %+v", estimate)
	}
	if estimate.PriceTableVersion == "" {
		t.Fatal("estimate must carry a price_table_version")
	}
	var egressLine *change.CostLineItem
	for index := range estimate.Breakdown {
		if estimate.Breakdown[index].Driver == "download egress" {
			egressLine = &estimate.Breakdown[index]
		}
	}
	if egressLine == nil {
		t.Fatal("expected a download egress line item")
	}
	if egressLine.SKU == "" || egressLine.Currency == "" || egressLine.Region == "" || egressLine.SourceURL == "" || egressLine.AsOf == "" {
		t.Fatalf("an ESTIMATED line item must be traceable to sku/region/currency/source_url/as_of: %+v", egressLine)
	}
	if len(estimate.Assumptions) == 0 {
		t.Fatal("an ESTIMATED estimate must record its assumptions")
	}
	// 2 TB == 2048 GiB by this parser's binary reading of GB-family units.
	if *estimate.ParsedVolumeGiB != 2048 {
		t.Fatalf("expected 2 TB to parse as 2048 GiB, got %v", *estimate.ParsedVolumeGiB)
	}
}

func TestEstimateOptionCostUnknownPriceProducesUnknownLine(t *testing.T) {
	original := change.PriceTable["gcs-network-egress-worldwide-tier1"]
	defer func() { change.PriceTable["gcs-network-egress-worldwide-tier1"] = original }()
	unknownPrice := original
	unknownPrice.Known = false
	change.PriceTable["gcs-network-egress-worldwide-tier1"] = unknownPrice

	estimate := change.EstimateOptionCost("managed_storage_with_signed_urls", storageConstraints("500 GB"))
	if estimate.Status != change.CostUnknown {
		t.Fatalf("expected UNKNOWN once the price is unverified, got %+v", estimate)
	}
	if estimate.MonthlyLow != 0 || estimate.MonthlyBase != 0 || estimate.MonthlyHigh != 0 {
		t.Fatalf("no amount may be invented once the price is unverified: %+v", estimate)
	}
}

func TestParseMonthlyVolumeGiB(t *testing.T) {
	cases := []struct {
		raw     string
		wantGiB float64
		wantOK  bool
	}{
		{"500 GB", 500, true},
		{"500GB", 500, true},
		{"500 GB/month", 500, true},
		{"1.5 TB per month", 1536, true},
		{"2,000 GiB", 2000, true},
		{"2 TiB", 2048, true},
		{"1024 MiB", 1, true},
		{"500 reviews", 0, false},
		{"10000 uploads", 0, false},
		{"", 0, false},
		{"   ", 0, false},
		{"-5 GB", 0, false},
		{"GB 500", 0, false},
	}
	for _, testCase := range cases {
		gib, ok := change.ParseMonthlyVolumeGiB(testCase.raw)
		if ok != testCase.wantOK {
			t.Fatalf("ParseMonthlyVolumeGiB(%q) ok = %v, want %v", testCase.raw, ok, testCase.wantOK)
		}
		if ok && gib != testCase.wantGiB {
			t.Fatalf("ParseMonthlyVolumeGiB(%q) = %v, want %v", testCase.raw, gib, testCase.wantGiB)
		}
	}
}
