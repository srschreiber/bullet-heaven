package model

import "math"

type Vec2 struct{ X, Y float32 }

var Vec2Zero = &Vec2{0, 0}

func (v *Vec2) Norm() *Vec2 {
	m := math.Hypot(float64(v.X), float64(v.Y))
	if m == 0 {
		return Vec2Zero
	}
	return &Vec2{v.X / float32(m), v.Y / float32(m)}
}
func (v *Vec2) Add(u *Vec2) *Vec2   { return &Vec2{v.X + u.X, v.Y + u.Y} }
func (v *Vec2) Sub(u *Vec2) *Vec2   { return &Vec2{v.X - u.X, v.Y - u.Y} }
func (v *Vec2) Mul(s float32) *Vec2 { return &Vec2{v.X * s, v.Y * s} }
func (v *Vec2) Distance(u *Vec2) float32 {
	return float32(math.Hypot(float64(v.X-u.X), float64(v.Y-u.Y)))
}
func (v *Vec2) Length() float32 {
	return float32(math.Hypot(float64(v.X), float64(v.Y)))
}

func (v Vec2) IsInBounds(w, h int, buffer int) bool {
	return v.X >= float32(buffer) && v.X < float32(w-buffer) &&
		v.Y >= float32(buffer) && v.Y < float32(h-buffer)
}
