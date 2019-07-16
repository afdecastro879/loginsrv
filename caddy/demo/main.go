package main

import (
	_ "github.com/BTBurke/caddy-jwt"
	_ "github.com/afdecastro/loginsrv/caddy"
	"github.com/mholt/caddy/caddy/caddymain"
)

func main() {
	caddymain.Run()
}
