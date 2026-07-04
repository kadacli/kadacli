package models

// ResolvedTarget is a provider-tagged wrapper around a fully resolved backup target.
// Payload holds the provider-specific resolved struct (e.g. azure.ResolvedVM).
// Consumers switch on Provider and type-assert Payload.
type ResolvedTarget struct {
	Provider string
	Payload  any
}
