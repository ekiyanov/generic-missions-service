package main

import (
	"fmt"
	"net/http"

	"github.com/ets/missions"

	"github.com/rs/cors"
)

func main() {
	fmt.Println("missions server for enter the sphere.")

	mux := missions.HttpMux("/api/missionsv1", missions.ClaimCallback)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})

	http.ListenAndServe(":8081", c.Handler(mux))

}
