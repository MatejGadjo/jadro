// Command vapidgen prints a fresh VAPID keypair for Web Push.
//
// Run once and copy the values into the deployment environment:
//
//	go run ./cmd/vapidgen      (or: make vapid)
//
// The public key is also handed to the browser (not a secret); the private key
// must stay server-side. Keys are persistent — regenerating them invalidates
// every existing push subscription, so generate once and keep them.
package main

import (
	"fmt"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}
	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", pub)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", priv)
	fmt.Println("VAPID_SUBJECT=mailto:kontakt@jadro.mk")
}
