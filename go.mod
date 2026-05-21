module github.com/waffo-com/waffo-pancake-sdk-go

go 1.22

// v0.1.0 was published with a checksum that does not match the current tag
// content (released during a tag-rewrite window that overlapped with
// sum.golang.org caching). Use v0.1.1 or later instead.
retract v0.1.0

// v0.3.0 shipped CreateCheckoutSessionParams.OrderMerchantExternalId (lowercase
// d), inconsistent with every other ID-suffixed field in the package, which
// broke compilation for callers referencing the canonical OrderMerchantExternalID.
// Use v0.3.1 or later.
retract v0.3.0
