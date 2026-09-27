package change

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// EstimateOptionCost is the deterministic cost calculator (VS-004 / P0-C).
// Same optionID + pricingRef + business constraints always produce the same
// CostEstimate: it calls no LLM, reads no clock and makes no network call.
// An advisor (RuleAdvisor or GeminiAdvisor) only names CostDrivers in words;
// this function is the only place a monthly amount is computed, and Service
// runs every proposed option through it before returning ChangeVersion.
//
// pricingRef is Option.PricingRef. When it is set, it takes priority over
// optionID for deciding whether (and how) to price the candidate — this is
// the only path a Gemini candidate can be priced through, since advisor.go
// may rewrite its own id to gemini_option_N. When pricingRef is empty (the
// RuleAdvisor path, unchanged from before pricing_ref existed), the original
// optionID match against costableOptions is used exactly as before. Neither
// path ever falls back to a fuzzy/best-effort match: an empty, "none" or
// unrecognised pricingRef is UNKNOWN with a reason naming the bad value, not
// guessed or matched loosely against costableOptions.
func EstimateOptionCost(optionID string, pricingRef string, businessConstraints map[string]string) CostEstimate {
	priceKey := optionID
	usingPricingRef := pricingRef != ""
	if usingPricingRef {
		priceKey = pricingRef
	}
	if !costableOptions[priceKey] {
		if usingPricingRef {
			return CostEstimate{
				Status:            CostUnknown,
				Reason:            fmt.Sprintf("pricing_ref %q is not a priced SKU in price_table_version %s", pricingRef, PriceTableVersion),
				PriceTableVersion: PriceTableVersion,
			}
		}
		return CostEstimate{
			Status:            CostUnknown,
			Reason:            fmt.Sprintf("no priced SKU is mapped to option %q in price_table_version %s", optionID, PriceTableVersion),
			PriceTableVersion: PriceTableVersion,
		}
	}

	raw := businessConstraints["expected_monthly_volume"]
	volumeGiB, ok := ParseMonthlyVolumeGiB(raw)
	if !ok {
		return CostEstimate{
			Status:            CostUnknown,
			Reason:            fmt.Sprintf("expected_monthly_volume %q is not a recognised \"<number> <GB|GiB|TB|TiB|MB|MiB>\" amount", raw),
			PriceTableVersion: PriceTableVersion,
		}
	}

	// Only the download-egress driver is priced in this P0-C slice: the
	// option's own summary already states that retention and access-pattern
	// inputs are needed before stored bytes or metadata operations can be
	// sized, and neither is a required business constraint today (see
	// requiredConstraints in service.go). Inventing a retention assumption
	// to price storage would contradict that documented unknown, so storage
	// and metadata stay UNKNOWN line items instead.
	storageItem := CostLineItem{
		Driver: "stored bytes over retention", Status: CostUnknown,
		Reason: "retention period is not a declared business constraint; storage duration cannot be sized",
	}
	metadataItem := CostLineItem{
		Driver: "metadata reads/writes", Status: CostUnknown,
		Reason: fmt.Sprintf("no priced SKU for Class A/B operations in price_table_version %s", PriceTableVersion),
	}

	egress := PriceTable["gcs-network-egress-worldwide-tier1"]
	var egressItem CostLineItem
	var low, base, high float64
	if !egress.Known {
		egressItem = CostLineItem{
			Driver: "download egress", SKU: egress.SKU, Status: CostUnknown,
			Reason: "no verified unit price recorded for " + egress.SKU,
		}
	} else {
		low = round2(volumeGiB * LowScenarioMultiplier * egress.UnitPrice)
		base = round2(volumeGiB * BaseScenarioMultiplier * egress.UnitPrice)
		high = round2(volumeGiB * HighScenarioMultiplier * egress.UnitPrice)
		egressItem = CostLineItem{
			Driver: "download egress", SKU: egress.SKU, Status: CostEstimated,
			Low: low, Base: base, High: high,
			Currency: egress.Currency, Unit: egress.Unit, Region: egress.Region,
			SourceURL: egress.SourceURL, AsOf: egress.AsOf,
			Reason: fmt.Sprintf("parsed_volume=%.4f GiB/month x scenario multiplier x unit price (%s)", volumeGiB, egress.SKU),
		}
	}

	parsedVolume := volumeGiB
	estimate := CostEstimate{
		PriceTableVersion: PriceTableVersion,
		ParsedVolumeGiB:   &parsedVolume,
		Breakdown:         []CostLineItem{egressItem, storageItem, metadataItem},
		Assumptions: []string{
			fmt.Sprintf("expected_monthly_volume is read as %.4f GiB/month of signed-URL download egress, not stored bytes", volumeGiB),
			fmt.Sprintf("low/base/high scenarios apply named multipliers (%.1fx / %.1fx / %.1fx) around the single reported estimate to model input uncertainty, not measured usage variance", LowScenarioMultiplier, BaseScenarioMultiplier, HighScenarioMultiplier),
			"stored bytes and metadata operations stay UNKNOWN until retention and access-pattern inputs are declared",
		},
	}
	if egressItem.Status == CostEstimated {
		estimate.Status = CostEstimated
		estimate.Currency = egress.Currency
		estimate.MonthlyLow, estimate.MonthlyBase, estimate.MonthlyHigh = low, base, high
		estimate.Region = egress.OriginRegion
	} else {
		estimate.Status = CostUnknown
		estimate.Reason = egressItem.Reason
	}
	// CoveredDrivers/UnpricedDrivers are read straight off Breakdown's own
	// Status field, never decided independently of it, so the two views of
	// "what did the calculator price" can never disagree. Only populated when
	// the estimate itself is ESTIMATED: when it is UNKNOWN there is no partial
	// coverage to disclose, only the one Reason.
	if estimate.Status == CostEstimated {
		for _, item := range estimate.Breakdown {
			if item.Status == CostEstimated {
				estimate.CoveredDrivers = append(estimate.CoveredDrivers, item.Driver)
			} else {
				estimate.UnpricedDrivers = append(estimate.UnpricedDrivers, item.Driver)
			}
		}
	}
	return estimate
}

// costableOptions names the option IDs this P0-C calculator can price —
// matched against optionID on the RuleAdvisor path, or against
// Option.PricingRef on any other path (see EstimateOptionCost). It is also
// the fixed choice list GeminiAdvisor's prompt and response schema constrain
// pricing_ref to (see pricingRefChoices in advisor.go), so the two can never
// drift into listing two different sets of priceable options. Any option
// this map has no key for is reported UNKNOWN, never guessed.
var costableOptions = map[string]bool{
	"managed_storage_with_signed_urls": true,
}

// Scenario multipliers model uncertainty in a single reported
// expected_monthly_volume estimate, not usage growth over time or measured
// variance: P0 has no historical Change volume data yet (DR-006 unknowns).
// low/high widen a deliberately simple +/-2x band around the one number a
// business owner supplies; revisit once real Candidate/Release volume
// evidence exists (see roadmap.md#7, G5 baseline measurement).
const (
	LowScenarioMultiplier  = 0.5
	BaseScenarioMultiplier = 1.0
	HighScenarioMultiplier = 2.0
)

// volumePattern matches a leading "<number><space><unit>" amount, e.g.
// "500 GB", "500GB/month", "1.5 TB per month", "2,000 GiB". It is
// deliberately conservative: no locale-specific decimal/thousands handling
// beyond a plain "," grouping separator, no unit inference from context, and
// no partial credit for a number with no recognised unit. Anything after the
// unit (such as "/month" or "per month") is accepted but not parsed, since
// expected_monthly_volume is already defined as a monthly figure.
var volumePattern = regexp.MustCompile(`(?i)^\s*([0-9][0-9,]*(?:\.[0-9]+)?)\s*(gib|gb|tib|tb|mib|mb)\b`)

// unitToGiB converts a recognised unit to gibibytes. Cloud Storage bills in
// JEDEC binary GB (i.e. GiB; see the "Pricing notes" section of
// https://cloud.google.com/storage/pricing), so GB/TB/MB here are read as
// the same binary units as GiB/TiB/MiB rather than as decimal SI units; the
// parser never silently mixes a 1000-based and 1024-based reading.
func unitToGiB(amount float64, unit string) float64 {
	switch strings.ToLower(unit) {
	case "gib", "gb":
		return amount
	case "tib", "tb":
		return amount * 1024
	case "mib", "mb":
		return amount / 1024
	default:
		return 0
	}
}

// ParseMonthlyVolumeGiB parses a business_constraints.expected_monthly_volume
// string into gibibytes/month. ok is false whenever the string does not
// start with a plain "<number> <GB-family unit>" amount (e.g. "500 reviews"
// or "10000 uploads", both real values already accepted elsewhere in this
// package as request-level business constraints) — the caller must then
// treat the whole cost result as UNKNOWN rather than guess at intent.
func ParseMonthlyVolumeGiB(raw string) (gib float64, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	match := volumePattern.FindStringSubmatch(trimmed)
	if match == nil {
		return 0, false
	}
	numeric := strings.ReplaceAll(match[1], ",", "")
	amount, err := strconv.ParseFloat(numeric, 64)
	if err != nil || amount < 0 {
		return 0, false
	}
	return unitToGiB(amount, match[2]), true
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
