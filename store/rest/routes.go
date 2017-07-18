package rest

import (
	"net/http"
	"github.com/zero-os/0-stor/store/rest/models"
	"github.com/justinas/alice"
)

type HttpRoutes struct{}

type HttpRouteEntry struct{
	Path string
	Handler func(http.ResponseWriter, *http.Request)
	Methods []string
	Middlewares []alice.Constructor
}

type MiddlewareEntry struct{
	Middlewares []alice.Constructor
}


func (h HttpRoutes) GetRoutes(i NamespacesInterface)[]HttpRouteEntry {
	db := i.DB()
	return []HttpRouteEntry{
		{

			Path:    "/namespaces/{nsid}/acl",
			Handler: i.nsidaclPost,
			Methods: []string{"POST"},
			Middlewares: []alice.Constructor{
				NewReservationValidMiddleware(db).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},

		{
			Handler: i.DeleteObject,
			Path:    "/namespaces/{nsid}/objects/{id}",
			Methods: []string{"DELETE"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // At least user should have Delete permissions
					Read:   false,
					Write:  false,
					Delete: true,
					Admin:  false,
				}).Handler,
				NewReservationValidMiddleware(i.DB()).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},

		{
			Handler: i.HeadObject,
			Path: "/namespaces/{nsid}/objects/{id}",
			Methods: []string{"HEAD"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // At least user should have Read permissions
					Read:   true,
					Write:  false,
					Delete: false,
					Admin:  false,
				}).Handler,
				NewReservationValidMiddleware(i.DB()).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},

		{
			Path: "/namespaces/{nsid}/objects/{id}",
			Handler: i.GetObject,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // At least user should have Read permissions
					Read:   true,
					Write:  false,
					Delete: false,
					Admin:  false,
				}).Handler,
				NewReservationValidMiddleware(i.DB()).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},

		{
			Path: "/namespaces/{nsid}/objects",
			Handler: i.Listobjects,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // At least user should have Read permissions
					Read:   true,
					Write:  false,
					Delete: false,
					Admin:  false,
				}).Handler,
				NewReservationValidMiddleware(i.DB()).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},

		{
			Path: "/namespaces/{nsid}/objects",
			Handler: i.Createobject,
			Methods: []string{"POST"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // At least user should have Write permissions
					Read:   false,
					Write:  true,
					Delete: false,
					Admin:  false,
				}).Handler,
				NewReservationValidMiddleware(i.DB()).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},
		//
		{
			Path: "/namespaces/{nsid}/reservation/{id}",
			Handler: i.nsidreservationidGet,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},

		{
			Path: "/namespaces/{nsid}/reservation",
			Handler: i.ListReservations,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},

		{
			Path: "/namespaces/{nsid}/reservation",
			Handler: i.CreateReservation,
			Methods: []string{"POST"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},
		{
			Path: "/namespaces/{nsid}/stats",
			Handler: i.StatsNamespace,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewDataTokenValidMiddleware(models.ACLEntry{ // Admin permissions
					Read:   true,
					Write:  true,
					Delete: true,
					Admin:  true,
				}).Handler,
				NewNamespaceStatMiddleware(db).Handler,
			},
		},
		{
			Path:  "/namespaces/stats",
			Handler: i.UpdateStoreStats,
			Methods: []string{"POST"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},
		{
			Path: "/namespaces/stats",
			Handler: i.GetStoreStats,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},
		{
			Path: "/namespaces/{nsid}",
			Handler: i.Deletensid,
			Methods: []string{"DELETE"},
			Middlewares: []alice.Constructor{
				NewNamespaceStatMiddleware(db).Handler,
			},
		},
		{
			Path: "/namespaces/{nsid}",
			Handler: i.Getnsid,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewNamespaceStatMiddleware(db).Handler,
			},
		},
		{
			Path: "/namespaces",
			Handler: i.Listnamespaces,
			Methods: []string{"GET"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},
		{
			Path: "/namespaces",
			Handler: i.Createnamespace,
			Methods: []string{"POST"},
			Middlewares: []alice.Constructor{
				NewOauth2itsyouonlineMiddleware([]string{"user:name"}).Handler,
			},
		},
	}



}
