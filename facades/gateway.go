package facades

import (
	"log"

	"github.com/goravel/gateway"
	"github.com/goravel/gateway/contracts"
)

func Gateway() contracts.Gateway {
	instance, err := gateway.App.Make(gateway.Binding)
	if err != nil {
		log.Println(err)
		return nil
	}

	return instance.(contracts.Gateway)
}
