package utils

// Helper functions for pointer conversions
func IntPtr(i int) *int             { return &i }
func Uint64Ptr(u uint64) *uint64    { return &u }
func StringPtr(s string) *string    { return &s }
func Float64Ptr(f float64) *float64 { return &f }
