package secrets

// Store defines the interface for working with secret sources.
type Store interface {
	// Resolve a credential's value from a secret store
	// - keyName is name of the key where the secret can be found.
	// - keyValue is the value of the key.
	// Examples:
	// - keyName=env, keyValue=CONN_STRING
	// - keyName=key, keyValue=conn-string
	// - keyName=path, keyValue=/tmp/connstring.txt
	Resolve(keyName string, keyValue string) (string, error)
}
