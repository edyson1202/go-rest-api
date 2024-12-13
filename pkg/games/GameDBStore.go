package games

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	host     = "postgres"
	port     = 5432
	user     = "postgres"
	password = "mysecretpassword"
	dbname   = "go-dev-database"
)

var (
	NotFoundErr = errors.New("not found")
)

type MemStore struct {
	db *sql.DB
}

func NewMemStore() *MemStore {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// the Open function does not actually open a connection to the database, it only validates the connection string
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	//defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	return &MemStore{
		db,
	}
}

func (m MemStore) Add(game Game) error {
	_, err := m.db.Exec(sqlInsertStatement, game.ReleaseYear, game.Name, game.Category)
	if err != nil {
		panic(err)
	}

	//	id := 0
	//	err = db.QueryRow(sqlStatement, 2008, "Red Dead Redemption", "Adventure").Scan(&id)
	//	if err != nil {
	//		panic(err)
	//	}
	//	fmt.Println("New record ID is:", id)
	//

	return nil
}

func (m MemStore) Get(id int) (Game, error) {
	var game Game

	row := m.db.QueryRow(sqlGetByIdStatement, id)
	switch err := row.Scan(&game.Id, &game.ReleaseYear, &game.Name, &game.Category); err {
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
	case nil:
		fmt.Println(game)
		return game, nil
	default:
		panic(err)
	}

	return Game{}, NotFoundErr
}

func (m MemStore) List(page Page) ([]Game, error) {
	rows, err := m.db.Query(sqlGetAllStatement, page.Size, page.Size*page.Page)
	if err != nil {
		// handle this error better than this
		panic(err)
	}
	defer rows.Close()

	var list []Game
	for rows.Next() {
		var id int
		var firstName string
		var game Game
		err = rows.Scan(&game.Id, &game.ReleaseYear, &game.Name, &game.Category)
		if err != nil {
			// handle this error
			panic(err)
		}
		fmt.Println(id, firstName)
		list = append(list, game)
	}
	// get any error encountered during iteration
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return list, nil
}

func (m MemStore) Update(id int, game Game) error {
	res, err := m.db.Exec(sqlUpdateStatement, id, game.ReleaseYear, game.Name, game.Category)
	if err != nil {
		panic(err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}
	fmt.Println("Rows affected: ")
	fmt.Println(count)

	if count != 0 {
		return nil
	}

	return NotFoundErr
}

func (m MemStore) Remove(id int) error {
	_, err := m.db.Exec(sqlDeleteStatement, id)
	if err != nil {
		panic(err)
	}

	return nil
}

//const sqlInsertStatement = `
//INSERT INTO games (release_year, name, category)
//VALUES ($1, $2, $3)
//RETURNING id`
//const sqlGetAllStatement = `SELECT id, release_year, name, category FROM games LIMIT $1;`
//const sqlGetByIdStatement = `SELECT id, release_year, name, category FROM games WHERE id=$1;`
//const sqlUpdateStatement = `
//UPDATE games
//SET release_year = $2, name = $3, category = $4
//WHERE id = $1;`
//const sqlDeleteStatement = `
//DELETE FROM games
//WHERE id = $1;`

const (
	sqlInsertStatement = `
INSERT INTO games (release_year, name, category)
VALUES ($1, $2, $3)
RETURNING id`
	sqlGetAllStatement  = `SELECT id, release_year, name, category FROM games LIMIT $1 OFFSET $2;`
	sqlGetByIdStatement = `SELECT id, release_year, name, category FROM games WHERE id=$1;`
	sqlUpdateStatement  = `
UPDATE games
SET release_year = $2, name = $3, category = $4
WHERE id = $1;`
	sqlDeleteStatement = `
DELETE FROM games
WHERE id = $1;`
)
