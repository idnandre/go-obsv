package fiber

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func TraceMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		beforeNext := c.Route().Path
		beforePath := c.Path()
		err := c.Next()
		afterNext := c.Route().Path

		fmt.Println("before path ", beforePath)
		fmt.Println("before ", beforeNext)
		fmt.Println("after ", afterNext)

		return err
	}
}