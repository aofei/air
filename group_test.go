package air

import "testing"

func TestGroup(t *testing.T) {
	g := &Group{
		Prefix: "/group",
	}

	g.GET("/", nil)
	g.HEAD("/", nil)
	g.POST("/", nil)
	g.PUT("/", nil)
	g.PATCH("/", nil)
	g.DELETE("/", nil)
	g.CONNECT("/", nil)
	g.OPTIONS("/", nil)
	g.TRACE("/", nil)
	g.STATIC("/", "")
	g.FILE("/file", "")
}
