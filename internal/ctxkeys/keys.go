package ctxkeys

type siteConfigKey struct{}

var SiteConfig = siteConfigKey{}

type versionKey struct{}

// Version carries the current Detent release tag for the request.
var Version = versionKey{}
