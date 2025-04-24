package conatho

import (
	"bufio"
	"bytes"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"io"
	"slices"

	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xfmoulet/qoi"
	"golang.org/x/image/draw"
)

var ErrEntityNoImage = errors.New("entity has no image")

type Connection struct {
	ID       uuid.UUID
	Name     string
	Superior uuid.UUID
	Inferior uuid.UUID
}

type Entity struct {
	ID          uuid.UUID
	X           int32
	Y           int32
	Name        string
	Image       bool
	Connections []uuid.UUID // UUIDs of connections

	c *Conatho // So we can access the main object from the entity methods
}

type Conatho struct {
	sql *sql.DB

	Entities     map[uuid.UUID]*Entity
	EntitiesKeys []uuid.UUID

	Connections     map[uuid.UUID]*Connection
	ConnectionsKeys []uuid.UUID

	AttributeTypes map[int64]AttributeType
}

const ThumbnailWidth = int(150)
const ThumbnailHeight = int(150)

func New(filename string) (Conatho, error) {
	var c Conatho

	sql, err := sql.Open("sqlite3", "file:"+filename+"?cache=shared")
	if err != nil {
		return c, errors.New("could not open file")
	}
	c.sql = sql

	// Initialize if version not found
	var version int64
	row := c.sql.QueryRow("SELECT version FROM info")
	if err := row.Scan(&version); err != nil {
		err = c.init()
		if err != nil {
			return c, err
		}
	}

	c.Entities = make(map[uuid.UUID]*Entity)
	c.Connections = make(map[uuid.UUID]*Connection)
	c.AttributeTypes = make(map[int64]AttributeType)

	return c, nil
}

func (c *Conatho) init() error {
	_, err := c.sql.Exec(`
	    CREATE TABLE "info" (
	        "version" BIGINT NOT NULL
		);
		CREATE TABLE "entities" (
			"id"	BLOB NOT NULL UNIQUE,
			"name"	TEXT NOT NULL,
			"posx"	BIGINT NOT NULL,
			"posy"	BIGINT NOT NULL,
			"image"	DEFAULT FALSE
		);
		CREATE TABLE "connections" (
			"id"		 BLOB NOT NULL UNIQUE,
			"superior"	 BLOB NOT NULL,
			"inferior"	 BLOB NOT NULL,
			"name"		 TEXT DEFAULT ""
		);
		CREATE TABLE "attribute_types" (
			"id"		 INTEGER PRIMARY KEY AUTOINCREMENT,
			"name"		 TEXT NOT NULL,
			"datatype"	 INT NOT NULL
		);
		CREATE TABLE "attributes" (
			"id"		INTEGER NOT NULL,
			"entity"	BLOB NOT NULL,
			"type"		INT NOT NULL,
			"num"		INT,
			"str"		TEXT,
			"data"		BLOB,
			PRIMARY KEY("id" AUTOINCREMENT),
			FOREIGN KEY("type") REFERENCES "attribute_types"("id")
		);
		CREATE TABLE "images" (
			"id"	BLOB NOT NULL,
			"image"	BLOB NOT NULL,
			"thumbnail"	BLOB NOT NULL,
			FOREIGN KEY("id") REFERENCES "entities"("id")
		);
	`)
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("could not initialise file")
	}

	_, err = c.sql.Exec("INSERT INTO info (version) VALUES (?)", 0)
	if err != nil {
		return err
	}

	return nil
}

func (c *Conatho) CreateEntity(posX, posY int32, name string) (Entity, error) {
	e := Entity{
		c:     c,
		ID:    uuid.New(),
		X:     posX,
		Y:     posY,
		Name:  name,
		Image: false,
	}

	id, err := e.ID.MarshalBinary()
	if err != nil {
		return e, err
	}

	_, err = c.sql.Exec("INSERT INTO entities (id, name, posx, posy) VALUES (?, ?, ?, ?)", id, e.Name, e.X, e.Y)
	if err != nil {
		return e, err
	}

	c.Entities[e.ID] = &e
	c.EntitiesKeys = append(c.EntitiesKeys, e.ID)

	return e, nil
}

func (c *Conatho) generateEntitiesKeys() {
	c.EntitiesKeys = make([]uuid.UUID, 0, len(c.Entities))
	for k := range c.Entities {
		c.EntitiesKeys = append(c.EntitiesKeys, k)
	}
}

func (c *Conatho) generateConnectionsKeys() {
	c.ConnectionsKeys = make([]uuid.UUID, 0, len(c.Connections))
	for k := range c.Connections {
		c.ConnectionsKeys = append(c.ConnectionsKeys, k)
	}
}

func (e *Entity) EntityAddImage(imageReader io.Reader) error {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return err
	}

	// Load image
	img, format, err := image.Decode(imageReader)
	if err != nil {
		return err
	}
	fmt.Println(format)

	// Save to buffer as QOI
	var qoiImg bytes.Buffer
	qoiImgWriter := bufio.NewWriter(&qoiImg)
	err = qoi.Encode(qoiImgWriter, img)
	if err != nil {
		panic(err)
	}

	bounds := img.Bounds()
	width := float32(bounds.Max.X)
	height := float32(bounds.Max.Y)
	// bounds.Max.X
	var rect image.Rectangle
	if width > height {
		newWidth := ThumbnailWidth
		newHeight := int(height / width * float32(ThumbnailWidth))
		rect = image.Rect(
			0,
			(ThumbnailHeight-newHeight)/2,
			newWidth,
			ThumbnailHeight-((ThumbnailHeight-newHeight)/2))
	} else if height > width {
		newHeight := ThumbnailHeight
		newWidth := int(width / height * float32(ThumbnailHeight))
		x0 := (ThumbnailWidth - newWidth) / 2
		y0 := 0
		x1 := ((ThumbnailWidth - newWidth) / 2) + newWidth
		y1 := newHeight
		fmt.Println(width, height)
		fmt.Println(newWidth, newHeight)
		fmt.Println(x0, y0, x1, y1)

		rect = image.Rect(
			(ThumbnailWidth-newWidth)/2,
			0,
			((ThumbnailWidth-newWidth)/2)+newWidth,
			newHeight)
	} else {
		rect = image.Rect(0, 0, 150, 150)
	}

	dst := image.NewRGBA(image.Rect(0, 0, ThumbnailWidth, ThumbnailHeight))
	draw.NearestNeighbor.Scale(dst, rect, img, img.Bounds(), draw.Over, nil)

	var qoiImgThumbnail bytes.Buffer
	qoiImgThumbnailWriter := bufio.NewWriter(&qoiImgThumbnail)

	err = qoi.Encode(qoiImgThumbnailWriter, dst)
	if err != nil {
		panic(err)
	}

	if !e.Image {
		_, err = e.c.sql.Exec("INSERT INTO images (id, image, thumbnail) VALUES (?, ?, ?)", id, qoiImg.Bytes(), qoiImgThumbnail.Bytes())
		if err != nil {
			return err
		}

		_, err = e.c.sql.Exec("UPDATE entities SET image = TRUE WHERE id = ?", id)
		if err != nil {
			return err
		}
	} else {
		_, err = e.c.sql.Exec("UPDATE images SET image = ?, thumbnail = ? WHERE id = ?", qoiImg.Bytes(), qoiImgThumbnail.Bytes(), id)
		if err != nil {
			return err
		}
	}

	e.Image = true

	return nil
}

func (e *Entity) EntityGetThumbnail() ([]byte, error) {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if !e.Image {
		return nil, ErrEntityNoImage
	}

	var img []byte
	row := e.c.sql.QueryRow("SELECT thumbnail FROM images WHERE id = ?", id)
	if err := row.Scan(&img); err != nil {
		return nil, err
	}

	return img, nil
}

func (e *Entity) EntityGetImage() ([]byte, error) {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if !e.Image {
		return nil, ErrEntityNoImage
	}

	var img []byte
	row := e.c.sql.QueryRow("SELECT image FROM images WHERE id = ?", id)
	if err := row.Scan(&img); err != nil {
		return nil, err
	}

	return img, nil
}

type Datatype int

const (
	DatatypeNumber Datatype = iota
	DatatypeString
	DatatypeData
)

type AttributeType struct {
	Name string
	Type Datatype
}

func (c *Conatho) GetAttributeTypes() error {
	c.AttributeTypes = make(map[int64]AttributeType)

	rows, err := c.sql.Query(`
		SELECT id, name, datatype
		FROM attribute_types`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var a AttributeType
		err := rows.Scan(&id, &a.Name, &a.Type)
		if err != nil {
			return err
		}

		c.AttributeTypes[id] = a
	}
	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

func (c *Conatho) AddAttributeType(name string, dataType Datatype) (int64, error) {
	var id int64

	row := c.sql.QueryRow("INSERT INTO attribute_types (name, datatype) VALUES (?, ?) RETURNING id", name, int64(dataType))
	err := row.Scan(&id)
	if err != nil {
		return id, err
	}

	c.AttributeTypes[id] = AttributeType{
		Name: name,
		Type: dataType,
	}

	return id, nil
}

type Attribute struct {
	ID     int64
	Name   string
	Type   Datatype
	Number int64
	String string
	Data   []byte
}

func (e *Entity) GetAttributes() ([]Attribute, error) {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	attributes := []Attribute{}

	rows, err := e.c.sql.Query(`
		SELECT attributes.id, attribute_types.name, attribute_types.datatype, attributes.num, attributes.str, attributes.data
		FROM attributes
		LEFT JOIN attribute_types ON attributes.type = attribute_types.id
		WHERE attributes.entity = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var attribute Attribute
		var num sql.NullInt64
		var str sql.NullString
		var data []byte
		err := rows.Scan(&attribute.ID, &attribute.Name, &attribute.Type, &num, &str, &data)
		if err != nil {
			return nil, err
		}

		switch attribute.Type {
		case DatatypeNumber:
			if num.Valid {
				attribute.Number = num.Int64
			}
		case DatatypeString:
			if str.Valid {
				attribute.String = str.String
			}
		case DatatypeData:
			attribute.Data = data
		}

		attributes = append(attributes, attribute)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return attributes, nil
}

func (e *Entity) AddAttribute(attributeTypeID int64) (int64, error) {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return 0, err
	}

	_, ok := e.c.AttributeTypes[attributeTypeID]
	if !ok {
		return 0, errors.New("unknown type")
	}

	row := e.c.sql.QueryRow("INSERT INTO attributes (entity, type) VALUES (?, ?) RETURNING id", id, attributeTypeID)

	var attributeID int64
	err = row.Scan(&attributeID)
	if err != nil {
		return 0, err
	}

	return attributeID, nil
}

func (e *Entity) UpdateAttribute(attributeID int64, value interface{}) error {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return err
	}

	switch v := value.(type) {
	case int64:
		_, err = e.c.sql.Exec("UPDATE attributes SET num = ? WHERE entity = ? AND id = ?", v, id, attributeID)
	case string:
		_, err = e.c.sql.Exec("UPDATE attributes SET str = ? WHERE entity = ? AND id = ?", v, id, attributeID)
	case []byte:
		_, err = e.c.sql.Exec("UPDATE attributes SET data = ? WHERE entity = ? AND id = ?", v, id, attributeID)
	default:
		return errors.New("unsupported type")
	}

	if err != nil {
		return err
	}

	return nil
}

func (e *Entity) Delete() error {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = e.c.sql.Exec("DELETE FROM entities WHERE id = ?", id)
	if err != nil {
		return err
	}

	_, err = e.c.sql.Exec("DELETE FROM connections WHERE superior = ? OR inferior = ?", id, id)
	if err != nil {
		return err
	}

	// Loop through all connections
	for _, cID := range e.Connections {
		connection, ok := e.c.Connections[cID]
		if !ok {
			continue
		}

		// Check if the entity to be deleted is the superior or inferior
		var entity *Entity
		if connection.Superior == e.ID {
			entity, ok = e.c.Entities[connection.Inferior]
		} else if connection.Inferior == e.ID {
			entity, ok = e.c.Entities[connection.Superior]
		}
		if ok {
			i := slices.IndexFunc(entity.Connections, func(id uuid.UUID) bool {
				return id == cID
			})
			if i >= 0 {
				entity.Connections = removeFromSlice(entity.Connections, i)
			}
		}

		// Delete from ConnectionKeys
		delete(e.c.Connections, cID)
		i := slices.IndexFunc(e.c.ConnectionsKeys, func(id uuid.UUID) bool {
			return id == cID
		})
		if i >= 0 {
			e.c.ConnectionsKeys = removeFromSlice(e.c.ConnectionsKeys, i)
		}
	}

	delete(e.c.Entities, e.ID)
	e.c.generateEntitiesKeys()

	return nil
}

func (e *Entity) ConnectTo(inferior *Entity, connectionName string) error {
	if e.ID == inferior.ID {
		return errors.New("can not connect to itself")
	}
	connection := Connection{
		ID:       uuid.New(),
		Name:     connectionName,
		Superior: e.ID,
		Inferior: inferior.ID,
	}

	id, err := connection.ID.MarshalBinary()
	if err != nil {
		return err
	}

	superiorID, err := connection.Superior.MarshalBinary()
	if err != nil {
		return err
	}

	inferiorID, err := connection.Inferior.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = e.c.sql.Exec("INSERT INTO connections (id, superior, inferior, name) VALUES (?, ?, ?, ?)", id, superiorID, inferiorID, connection.Name)
	if err != nil {
		return err
	}

	e.c.Connections[connection.ID] = &connection
	e.c.ConnectionsKeys = append(e.c.ConnectionsKeys, connection.ID)

	e.Connections = append(e.Connections, connection.ID)
	inferior.Connections = append(inferior.Connections, connection.ID)

	return nil
}

func (e *Entity) UpdatePosition() error {
	id, err := e.ID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = e.c.sql.Exec("UPDATE entities SET posx = ?, posy = ? WHERE id = ?", e.X, e.Y, id)
	if err != nil {
		return err
	}
	return nil
}

func removeFromSlice[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}

func (c *Conatho) RemoveConnection(connection *Connection) error {
	id, err := connection.ID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = c.sql.Exec("DELETE FROM connections WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Remove from superior
	superior := c.Entities[connection.Superior]
	i := slices.IndexFunc(superior.Connections, func(id uuid.UUID) bool {
		return id == connection.ID
	})
	if i >= 0 {
		superior.Connections = removeFromSlice(superior.Connections, i)
	}

	// Remove from inferior
	inferior := c.Entities[connection.Inferior]
	i = slices.IndexFunc(inferior.Connections, func(id uuid.UUID) bool {
		return id == connection.ID
	})
	if i >= 0 {
		inferior.Connections = removeFromSlice(inferior.Connections, i)
	}

	// Remove from ConnectionsKeys
	i = slices.IndexFunc(c.ConnectionsKeys, func(id uuid.UUID) bool {
		return id == connection.ID
	})
	if i >= 0 {
		c.ConnectionsKeys = removeFromSlice(c.ConnectionsKeys, i)
	}

	delete(c.Connections, connection.ID)

	return nil
}

// Get all entities from file
func (c *Conatho) EntityGetAll() error {
	// Empty entities
	c.Entities = make(map[uuid.UUID]*Entity)

	rows, err := c.sql.Query(`
		SELECT id, name, posx, posy, image
		FROM entities
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		e := Entity{c: c}
		err := rows.Scan(&e.ID, &e.Name, &e.X, &e.Y, &e.Image)
		if err != nil {
			return err
		}

		c.Entities[e.ID] = &e
	}
	if err = rows.Err(); err != nil {
		return err
	}

	// Generate the keys so we can loop through easily
	c.generateEntitiesKeys()

	// Empty connections
	c.Connections = make(map[uuid.UUID]*Connection)

	rows, err = c.sql.Query(`
		SELECT id, superior, inferior, name
		FROM connections
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		connection := Connection{}
		err := rows.Scan(&connection.ID, &connection.Superior, &connection.Inferior, &connection.Name)
		if err != nil {
			return err
		}

		c.Connections[connection.ID] = &connection

		superior, ok := c.Entities[connection.Superior]
		if ok {
			superior.Connections = append(superior.Connections, connection.ID)
		}

		inferior, ok := c.Entities[connection.Inferior]
		if ok {
			inferior.Connections = append(inferior.Connections, connection.ID)
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}

	c.generateConnectionsKeys()

	return nil
}
