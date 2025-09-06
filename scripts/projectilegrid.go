/*
This file contains the ProjectileGrid struct and its methods for managing bullet positions and collisions.
*/
package scripts

type ProjectileCell struct {
	Projectiles []*Projectile
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
	if pg.Cells[cellX] != nil {
		return pg.Cells[cellX][cellY]
	} else {
		pg.Cells[cellX] = make(map[int]*ProjectileCell)
		pg.Cells[cellX][cellY] = &ProjectileCell{}
		return pg.Cells[cellX][cellY]
	}
}

func (pg *ProjectileGrid) MoveProjectile(p *Projectile) {
	oldCell := pg.ProjectileToCell[p]
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

func (pg *ProjectileGrid) GetSurroundingCells(pos Vec2, radius int) []*ProjectileCell {
	/*
		pos: The position to check for projectiles
		radius: The distance around the position to check
	*/
	var cells []*ProjectileCell
	radiusInCells := radius/pg.CellSize + 1
	centerX := int(pos.X) / pg.CellSize
	centerY := int(pos.Y) / pg.CellSize

	topLeftX := centerX - radiusInCells
	topLeftY := centerY - radiusInCells
	bottomRightX := centerX + radiusInCells
	bottomRightY := centerY + radiusInCells

	for x := topLeftX; x <= bottomRightX; x++ {
		for y := topLeftY; y <= bottomRightY; y++ {
			if pg.Cells[x] == nil {
				continue
			}

			if cell := pg.Cells[x][y]; cell != nil {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}
