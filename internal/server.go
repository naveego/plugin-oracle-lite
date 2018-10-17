package internal

import (
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"context"

	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/naveego/plugin-oracle/internal/pub"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

type Server struct {
	mu         *sync.Mutex
	log        hclog.Logger
	settings   *Settings
	db         *sql.DB
	publishing bool
	connected  bool
}

// NewServer creates a new publisher Host.
func NewServer(logger hclog.Logger) pub.PublisherServer {
	return &Server{
		mu:  &sync.Mutex{},
		log: logger,
	}
}

func (s *Server) Connect(ctx context.Context, req *pub.ConnectRequest) (*pub.ConnectResponse, error) {
	s.log.Debug("Connecting...")
	s.settings = nil
	s.connected = false

	settings := new(Settings)
	if err := json.Unmarshal([]byte(req.SettingsJson), settings); err != nil {
		return nil, errors.WithStack(err)
	}

	connectionString, err := settings.GetConnectionString()
	if err != nil {
		return nil, err
	}

	s.db, err = sql.Open("goracle", connectionString)
	if err != nil {
		return nil, errors.Errorf("could not open connection: %s", err)
	}

	err = s.db.Ping()

	if err != nil {
		return nil, errors.Errorf("could not ping database: %s", err)
	}

	// connection made and tested

	s.connected = true
	s.settings = settings

	s.log.Debug("Connect completed successfully.")

	return new(pub.ConnectResponse), err
}

func (s *Server) DiscoverShapes(ctx context.Context, req *pub.DiscoverShapesRequest) (*pub.DiscoverShapesResponse, error) {

	s.log.Debug("Handling DiscoverShapesRequest...")

	if !s.connected {
		return nil, errNotConnected
	}

	var shapes []*pub.Shape
	var err error

	if req.Mode == pub.DiscoverShapesRequest_ALL {
		s.log.Debug("Discovering all tables and views...")
		shapes, err = s.getAllShapesFromSchema()
		s.log.Debug("Discovered tables and views.", "count", len(shapes))

		if err != nil {
			return nil, errors.Errorf("could not load tables and views from SQL: %s", err)
		}
	} else {
		s.log.Debug("Refreshing schemas from request.", "count", len(req.ToRefresh))
		for _, s := range req.ToRefresh {
			shapes = append(shapes, s)
		}
	}

	resp := &pub.DiscoverShapesResponse{}

	wait := new(sync.WaitGroup)

	for i := range shapes {
		shape := shapes[i]
		// include this shape in wait group
		wait.Add(1)

		// concurrently get details for shape
		go func() {
			s.log.Debug("Getting details for discovered schema...", "id", shape.Id)
			err := s.populateShapeColumns(shape)
			if err != nil {
				s.log.With("shape", shape.Id).With("err", err).Error("Error discovering columns.")
				shape.Errors = append(shape.Errors, fmt.Sprintf("Could not discover columns: %s", err))
				goto Done
			}
			s.log.Debug("Got details for discovered schema.", "id", shape.Id)

			s.log.Debug("Getting count for discovered schema...", "id", shape.Id)
			shape.Count, err = s.getCount(shape)
			if err != nil {
				s.log.With("shape", shape.Id).With("err", err).Error("Error getting row count.")
				shape.Errors = append(shape.Errors, fmt.Sprintf("Could not get row count for shape: %s", err))
				goto Done
			}
			s.log.Debug("Got count for discovered schema.", "id", shape.Id, "count", shape.Count.String())

			if req.SampleSize > 0 {
				s.log.Debug("Getting sample for discovered schema...", "id", shape.Id, "size", req.SampleSize)
				publishReq := &pub.PublishRequest{
					Shape: shape,
					Limit: req.SampleSize,
				}
				records := make(chan *pub.Record)

				go func() {
					err = s.readRecords(ctx, publishReq, records)
				}()

				for record := range records {
					shape.Sample = append(shape.Sample, record)
				}

				if err != nil {
					s.log.With("shape", shape.Id).With("err", err).Error("Error collecting sample.")
					shape.Errors = append(shape.Errors, fmt.Sprintf("Could not collect sample: %s", err))
					goto Done
				}
				s.log.Debug("Got sample for discovered schema.", "id", shape.Id, "size", len(shape.Sample))
			}
		Done:
			wait.Done()
		}()
	}

	// wait until all concurrent shape details have been loaded
	wait.Wait()

	for _, shape := range shapes {
		resp.Shapes = append(resp.Shapes, shape)
	}

	sort.Sort(pub.SortableShapes(resp.Shapes))

	return resp, nil

}

var queryID int32 = 0

func (s *Server) executeQuery(query string) (*sql.Rows, error) {
	t := time.Now()
	id := atomic.AddInt32(&queryID, 1)
	log := s.log.With("id", id)
	log.With("query", query).Debug("Executing query...")
	r, err := s.db.Query(query)
	e := time.Since(t)
	log.With("elapsed", e.Seconds()).Debug("Query complete.")
	return r, err
}

func (s *Server) getAllShapesFromSchema() ([]*pub.Shape, error) {

	// This query gets all tables in all schemas, but excludes the built in
	// tables that are part of Oracle and its plugins.
	rows, err := s.executeQuery(`
SELECT OWNER, TABLE_NAME 
FROM ALL_TABLES
WHERE TABLESPACE_NAME NOT IN ('SYSTEM', 'SYSAUX', 'TEMP', 'UNDOTBS1')
`)

	if err != nil {
		return nil, errors.Errorf("could not list tables: %s", err)
	}

	var shapes []*pub.Shape

	for rows.Next() {
		shape := new(pub.Shape)

		var (
			schemaName string
			tableName  string
		)
		err = rows.Scan(&schemaName, &tableName)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		shape.Id = fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)
		shape.Name = fmt.Sprintf("%s.%s", schemaName, tableName)

		shapes = append(shapes, shape)
	}

	return shapes, nil
}

type columnInfo struct {
	ColumnName            string `sql:"COLUMN_NAME"`
	ParameterizedDataType string
	DataType              string `sql:"DATA_TYPE"`
	DataLength            *int64 `sql:"DATA_LENGTH"`
	DataPrecision         *int64 `sql:"DATA_PRECISION"`
	DataScale             *int64
	NullableChar           string
}
func (c columnInfo) Nullable() bool {
	return c.NullableChar == "Y"
}

var deparameterizer = regexp.MustCompile(`\(\d+\)`)

func (s *Server) populateShapeColumns(shape *pub.Shape) (error) {

	var columnInfos []columnInfo

	query := shape.Query
	if query == "" {
		segs := strings.SplitN(shape.Id, ".", 2)
		if len(segs) != 2 {
			return errors.Errorf("ID %q did not have owner segment", shape.Id)
		}
		owner, table := strings.Trim(segs[0], `"`), strings.Trim(segs[1], `"`)
		query = fmt.Sprintf(`SELECT 
COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION, DATA_SCALE, NULLABLE
FROM ALL_TAB_COLUMNS WHERE OWNER = '%s' AND TABLE_NAME = '%s'`, owner, table)

		rows, err := s.executeQuery(query)
		if err != nil {
			return err
		}

		for rows.Next() {
			ci := columnInfo{}
			err = rows.Scan(&ci.ColumnName, &ci.DataType, &ci.DataLength, &ci.DataPrecision, &ci.DataScale, &ci.NullableChar)
			if err != nil {
				return err
			}
			ci.DataType = deparameterizer.ReplaceAllString(ci.DataType, "")
			columnInfos = append(columnInfos, ci)
		}

		if rows.Err() != nil {
			return rows.Err()
		}

	} else {
		metaQuery := fmt.Sprintf(`
SELECT SRC.* 
FROM (%s) SRC
WHERE rownum <= 1
ORDER BY rownum`, strings.Trim(query, ";"))

		rows, err := s.executeQuery(metaQuery)

		if err != nil {
			return errors.Errorf("error executing query %q: %v", metaQuery, err)
		}

		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			return errors.WithMessage(err, "could not get column types")
		}
		for _, ct := range columnTypes {
			ci := columnInfo{
				ColumnName: ct.Name(),
			}

			if n, ok := ct.Nullable(); ok && n{
				ci.NullableChar = "Y"
			}

			if p, s, ok := ct.DecimalSize(); ok {
				ci.DataPrecision = &p
				ci.DataScale = &s
			}
			if l, ok := ct.Length(); ok {
				ci.DataLength = &l
			}
			dt := ct.DatabaseTypeName()
			ci.ParameterizedDataType = dt
			ci.DataType = deparameterizer.ReplaceAllString(dt, "")
			columnInfos = append(columnInfos, ci)
		}
	}

	unnamedColumnIndex := 0

	for _, m := range columnInfos {

		var property *pub.Property
		var propertyID string

		propertyName := m.ColumnName
		if propertyName == "" {
			propertyName = fmt.Sprintf("UNKNOWN_%d", unnamedColumnIndex)
			unnamedColumnIndex++
		}

		propertyID = fmt.Sprintf(`"%s"`, propertyName)

		for _, p := range shape.Properties {
			if p.Id == propertyID {
				property = p
				break
			}
		}
		if property == nil {
			property = &pub.Property{
				Id:   propertyID,
				Name: propertyName,
			}
			shape.Properties = append(shape.Properties, property)
		}

		dtn := m.DataType
		switch dtn {
		case "CHAR", "VARCHAR2", "NCHAR", "NVARCHAR2":
			if m.DataLength != nil {
				dtn = fmt.Sprintf("%s(%d)", dtn, *m.DataLength)
			}
		case "NUMBER":
			if m.DataPrecision != nil && m.DataScale != nil {
				dtn = fmt.Sprintf("%s(%d,%d)", dtn, *m.DataPrecision, *m.DataScale)
			}
		}

		property.TypeAtSource = dtn

		property.Type = convertSQLType(m)

		property.IsNullable = m.Nullable()
	}

	return nil
}

func (s *Server) PublishStream(req *pub.PublishRequest, stream pub.Publisher_PublishStreamServer) error {

	jsonReq, _ := json.Marshal(req)

	s.log.Debug("Got PublishStream request.", "req", string(jsonReq))

	if !s.connected {
		return errNotConnected
	}

	// if s.settings.PrePublishQuery != "" {
	// 	_, err := s.db.Exec(s.settings.PrePublishQuery)
	// 	if err != nil {
	// 		return errors.Errorf("error running pre-publish query: %s", err)
	// 	}
	// }

	var err error
	records := make(chan *pub.Record)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err = s.readRecords(ctx, req, records)
	}()

	for record := range records {
		sendErr := stream.Send(record)
		if sendErr != nil {
			cancel()
			err = sendErr
			break
		}
	}

	// if s.settings.PostPublishQuery != "" {
	// 	_, postPublishErr := s.db.Exec(s.settings.PostPublishQuery)
	// 	if postPublishErr != nil {
	// 		if err != nil {
	// 			postPublishErr = errors.Errorf("%s (publish had already stopped with error: %s)", postPublishErr, err)
	// 		}
	//
	// 		return errors.Errorf("error running post-publish query: %s", postPublishErr)
	// 	}
	// }

	return err
}

func (s *Server) Disconnect(context.Context, *pub.DisconnectRequest) (*pub.DisconnectResponse, error) {
	if s.db != nil {
		s.db.Close()
	}

	s.connected = false
	s.settings = nil
	s.db = nil

	return new(pub.DisconnectResponse), nil
}

func (s *Server) getCount(shape *pub.Shape) (*pub.Count, error) {

	cErr := make(chan error)
	cCount := make(chan int)

	go func() {
		defer close(cErr)
		defer close(cCount)

		query, err := buildQuery(&pub.PublishRequest{
			Shape: shape,
		})
		if err != nil {
			cErr <- err
			return
		}

		query = fmt.Sprintf("SELECT COUNT(1) FROM (%s) Q", strings.Trim(query, ";"))

		row := s.db.QueryRow(query)
		var count int
		err = row.Scan(&count)
		if err != nil {
			cErr <- fmt.Errorf("error from query %q: %s", query, err)
			return
		}

		cCount <- count
	}()

	select {
	case err := <-cErr:
		return nil, err
	case count := <-cCount:
		return &pub.Count{
			Kind:  pub.Count_EXACT,
			Value: int32(count),
		}, nil
	case <-time.After(time.Second):
		return &pub.Count{
			Kind: pub.Count_UNAVAILABLE,
		}, nil
	}
}

func (s *Server) readRecords(ctx context.Context, req *pub.PublishRequest, out chan<- *pub.Record) error {

	defer close(out)

	var err error
	var query string

	query, err = buildQuery(req)
	if err != nil {
		return errors.Errorf("could not build query: %v", err)
	}

	rows, err := s.executeQuery(query)
	if err != nil {
		return errors.Errorf("error executing query %q: %v", query, err)
	}

	properties := req.Shape.Properties
	valueBuffer := make([]interface{}, len(properties))
	mapBuffer := make(map[string]interface{}, len(properties))

	for rows.Next() {
		if ctx.Err() != nil || !s.connected {
			return nil
		}

		for i, p := range properties {
			switch p.Type {
			case pub.PropertyType_FLOAT:
				var x float64
				valueBuffer[i] = &x
			case pub.PropertyType_INTEGER:
				var x int64
				valueBuffer[i] = &x
			case pub.PropertyType_DECIMAL:
				var x string
				valueBuffer[i] = &x
			default:
				if strings.HasPrefix(p.TypeAtSource, "INTERVAL") {
					var x string
					valueBuffer[i] = &x
				} else {
					valueBuffer[i] = &valueBuffer[i]
				}
			}
		}
		err = rows.Scan(valueBuffer...)
		if err != nil {
			return errors.WithStack(err)
		}

		for i, p := range properties {
			value := valueBuffer[i]
			switch p.TypeAtSource {
			case "DATE", "TIMESTAMP":
				if t, ok := value.(time.Time); ok {
					// strip time zone error from pure date and
					// from timestamp without timezone
					value = t.Format("2006-01-02T15:04:05.999999999Z")
				}
			}

			mapBuffer[p.Id] = value
		}

		var record *pub.Record
		record, err = pub.NewRecord(pub.Record_UPSERT, mapBuffer)
		if err != nil {
			return errors.WithStack(err)
		}
		out <- record
	}

	if err == nil && rows.Err() != nil {
		err = errors.WithMessage(rows.Err(), "error while scanning data")
	}

	return err
}

func buildQuery(req *pub.PublishRequest) (string, error) {

	q := req.Shape.Query

	if q == "" {
		w := new(strings.Builder)
		w.WriteString("SELECT ")

		var selectors []string
		for _, p := range req.Shape.Properties {

			sel := p.Id
			switch p.TypeAtSource {
			case "XMLTYPE":
				sel = fmt.Sprintf(`XMLTYPE.getCLOBVal(%s)`, p.Id)
			default:
				if strings.HasPrefix(p.TypeAtSource, "INTERVAL") {
					sel = fmt.Sprintf(`TO_CHAR(%s)`, p.Id)
				}
			}

			selectors = append(selectors, sel)
		}
		columns := strings.Join(selectors, ", ")
		fmt.Fprintln(w, columns)
		fmt.Fprintln(w, "FROM ", req.Shape.Id)

		if len(req.Filters) > 0 {
			fmt.Fprintln(w, "WHERE")

			properties := make(map[string]*pub.Property, len(req.Shape.Properties))
			for _, p := range req.Shape.Properties {
				properties[p.Id] = p
			}

			var filters []string
			for _, f := range req.Filters {
				property, ok := properties[f.PropertyId]
				if !ok {
					continue
				}

				wf := new(strings.Builder)

				fmt.Fprintf(wf, "  %s ", f.PropertyId)
				switch f.Kind {
				case pub.PublishFilter_EQUALS:
					fmt.Fprint(wf, "= ")
				case pub.PublishFilter_GREATER_THAN:
					fmt.Fprint(wf, "> ")
				case pub.PublishFilter_LESS_THAN:
					fmt.Fprint(wf, "< ")
				default:
					continue
				}

				switch property.Type {
				case pub.PropertyType_INTEGER, pub.PropertyType_FLOAT:
					fmt.Fprintf(wf, "%v", f.Value)
				case pub.PropertyType_DATETIME, pub.PropertyType_DATE:
					fmt.Fprintf(wf, `to_timestamp('%s', 'yyyy-mm-dd"T"hh24:mi:ss"Z"TZH:TZM')`, f.Value)
				default:
					fmt.Fprintf(wf, "CAST('%s' as %s)", f.Value, property.TypeAtSource)
				}

				filters = append(filters, wf.String())
			}

			fmt.Fprintln(w, strings.Join(filters, "AND\n  "))

		}

		q = w.String()
	}

	if req.Limit > 0 {
		q = fmt.Sprintf(`SELECT SRC.* FROM (
%s
) SRC 
WHERE rownum <= %d `, q, req.Limit)
	}

	return q, nil
}

var errNotConnected = errors.New("not connected")

func convertSQLType(ci columnInfo) pub.PropertyType {

	typeName := strings.ToUpper(strings.Split(ci.DataType, "(")[0])

	switch typeName {
	case "DATE", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITH TIMEZONE", "TIMESTAMP WITH LOCAL TIME ZONE", "TIMESTAMP WITH LOCAL TIMEZONE":
		return pub.PropertyType_DATETIME
	case "NUMBER":

		if ci.DataScale != nil && ci.DataPrecision != nil {
			precision, scale := *ci.DataPrecision, *ci.DataScale
			if scale == 0 || scale == -127 {
				if precision <= 16 {
					return pub.PropertyType_INTEGER
				}
			}
		}

		return pub.PropertyType_DECIMAL
	case "FLOAT", "BINARY_FLOAT", "DOUBLE", "BINARY_DOUBLE":
		return pub.PropertyType_FLOAT
	case "BOOLEAN":
		return pub.PropertyType_BOOL
	case "BLOB":
		return pub.PropertyType_BLOB
	case "XMLTYPE":
		return pub.PropertyType_XML
	case "CLOB", "NCLOB":
		return pub.PropertyType_TEXT
	case "CHAR", "VARCHAR", "NCHAR", "NVARCHAR", "VARCHAR2", "NVARCHAR2":
		if ci.DataLength != nil {
			length := *ci.DataLength
			if length >= 1024 {
				return pub.PropertyType_TEXT
			}
		}

		return pub.PropertyType_STRING
	default:
		return pub.PropertyType_STRING
	}
}
