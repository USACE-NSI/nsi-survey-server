package config

type Config struct {
	SkipJWT       bool
	LambdaContext bool
	Ippk          string
	Port          string
	Aud           string
}
