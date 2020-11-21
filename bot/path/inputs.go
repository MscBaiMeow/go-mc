package path

// Inputs describes the desired movements of the player.
type Inputs struct {
	Yaw, Pitch float64

	ThrottleX, ThrottleZ float64

	Jump bool
}
