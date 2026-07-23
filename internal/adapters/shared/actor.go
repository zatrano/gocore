package shared

import "github.com/gofiber/fiber/v3"

// ActorFromCtx, oturum açmış kullanıcının kimliğini Locals'tan okur.
func ActorFromCtx(c fiber.Ctx) (id, role string) {
	id, _ = c.Locals(LocalUserID).(string)
	role, _ = c.Locals(LocalRole).(string)
	return id, role
}
