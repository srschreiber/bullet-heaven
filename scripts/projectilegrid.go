/*
This file contains the ProjectileGrid struct and its methods for managing bullet positions and collisions.
*/
package scripts

type ProjectileCell struct {
	Projectiles []*Projectile
	X           int
	Y           int
}

type ProjectileGrid struct {
	CellSize         int
	Cells            map[int]map[int]*ProjectileCell
	ProjectileToCell map[*Projectile]*ProjectileCell
}

func NewProjectileGrid(cellSize int) *ProjectileGrid {
	return &ProjectileGrid{
		CellSize:         cellSize,
		Cells:            make(map[int]map[int]*ProjectileCell),
		ProjectileToCell: make(map[*Projectile]*ProjectileCell),
	}
}

func (pg *ProjectileGrid) GetCell(pos *Vec2) *ProjectileCell {
	cellX := int(pos.X) / pg.CellSize
	cellY := int(pos.Y) / pg.CellSize
	if pg.Cells[cellX] != nil && pg.Cells[cellX][cellY] != nil {
		return pg.Cells[cellX][cellY]
	} else {
		newCell := &ProjectileCell{
			X: cellX,
			Y: cellY,
		}

		if pg.Cells[cellX] == nil {
			pg.Cells[cellX] = make(map[int]*ProjectileCell)
		}

		pg.Cells[cellX][cellY] = newCell
		return newCell
	}
}

func (pg *ProjectileGrid) MoveProjectile(p *Projectile, oldPos *Vec2) {
	oldCell := pg.GetCell(oldPos)
	newCell := pg.GetCell(p.Pos)
	if oldCell != newCell {
		pg.RemoveProjectile(p)
		pg.AddProjectile(p)
	}
}

func (pg *ProjectileGrid) AddProjectile(p *Projectile) {
	cell := pg.GetCell(p.Pos)
	if cell != nil {
		cell.Projectiles = append(cell.Projectiles, p)
		pg.ProjectileToCell[p] = cell
	}
}

func (pg *ProjectileGrid) RemoveProjectile(p *Projectile) {
	cell := pg.ProjectileToCell[p]
	if cell != nil {
		for i, proj := range cell.Projectiles {
			if proj == p {
				cell.Projectiles = append(cell.Projectiles[:i], cell.Projectiles[i+1:]...)
				break
			}
		}
		delete(pg.ProjectileToCell, p)
	}
}

func (pg *ProjectileGrid) GetSurroundingProjectiles(pos *Vec2, radius int) []*Projectile {
	/*
		pos: The position to check for projectiles
		radius: The distance around the position to check
	*/
	var projectiles []*Projectile

	centerCell := pg.GetCell(pos)

	radius = (radius / pg.CellSize) + 1

	l := centerCell.X - radius
	r := centerCell.X + radius
	t := centerCell.Y - radius
	b := centerCell.Y + radius

	for x := l; x <= r; x++ {
		if pg.Cells[x] == nil {
			continue
		}
		for y := t; y <= b; y++ {
			if cell := pg.Cells[x][y]; cell != nil {
				projectiles = append(projectiles, cell.Projectiles...)
			}
		}
	}

	return projectiles
}
