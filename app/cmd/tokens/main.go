// Generator tokenów JWT: tokens.json z N tożsamościami legalnymi i jednym tokenem napastnika.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"thesis/app/internal/security"
)

func main() {
	n := flag.Int("n", 100, "liczba tożsamości legalnych")
	out := flag.String("out", "loadtests/tokens.json", "plik wyjściowy")
	ttl := flag.Duration("ttl", 24*time.Hour, "ważność tokenów")
	flag.Parse()
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "thesis-secret"
	}
	j := security.NewJWT(secret)
	res := struct {
		Legit    []string `json:"legit"`
		Attacker string   `json:"attacker"`
	}{}
	for i := 1; i <= *n; i++ {
		t, err := j.Sign(fmt.Sprintf("user-%d", i), *ttl)
		if err != nil {
			log.Fatal(err)
		}
		res.Legit = append(res.Legit, t)
	}
	var err error
	if res.Attacker, err = j.Sign("attacker", *ttl); err != nil {
		log.Fatal(err)
	}
	b, _ := json.MarshalIndent(res, "", " ")
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("zapisano %d tokenów + napastnik do %s", *n, *out)
}
