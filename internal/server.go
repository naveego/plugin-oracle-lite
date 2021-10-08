package internal

import (
	"io"
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

const Custom = "Custom"

type Server struct {
	mu         *sync.Mutex
	log        hclog.Logger
	settings   *Settings
	db         *sql.DB
	publishing bool
	connected  bool

	WriteSettings *WriteSettings
	StoredProcedures []string
}

// NewServer creates a new publisher Host.
func NewServer(logger hclog.Logger) pub.PublisherServer {
	return &Server{
		mu:  &sync.Mutex{},
		log: logger,
	}
}

func (s *Server) Configure(ctx context.Context, req *pub.ConfigureRequest) (*pub.ConfigureResponse, error) {
	return nil, errors.New("Not supported.")
}

func (s *Server) ConnectSession(*pub.ConnectRequest, pub.Publisher_ConnectSessionServer) error {
	return errors.New("Not supported.")
}

func (s *Server) ConfigureConnection(ctx context.Context, req *pub.ConfigureConnectionRequest) (*pub.ConfigureConnectionResponse, error) {
	return nil, errors.New("Not supported.")
}

func (s *Server) ConfigureQuery(ctx context.Context, req *pub.ConfigureQueryRequest) (*pub.ConfigureQueryResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (s *Server) ConfigureRealTime(ctx context.Context, req *pub.ConfigureRealTimeRequest) (*pub.ConfigureRealTimeResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (s *Server) BeginOAuthFlow(ctx context.Context, req *pub.BeginOAuthFlowRequest) (*pub.BeginOAuthFlowResponse, error) {
	return nil, errors.New("Not supported.")
}

func (s *Server) CompleteOAuthFlow(ctx context.Context, req *pub.CompleteOAuthFlowRequest) (*pub.CompleteOAuthFlowResponse, error) {
	return nil, errors.New("Not supported.")
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
	s.StoredProcedures = nil
	s.StoredProcedures = append(s.StoredProcedures, Custom)

	var connectionResponse = new(pub.ConnectResponse)

	if s.settings.ShouldDiscoverWrite() {
		// get stored procedures
		rows, err := s.db.Query("SELECT owner, object_name FROM dba_objects WHERE object_type = 'PROCEDURE' AND oracle_maintained != 'Y' AND status = 'VALID'")
		if err != nil {
			connectionResponse.ConnectionError = fmt.Sprintf("could not read stored procedures from database: %s",err)
			return connectionResponse, nil
		}

		for rows.Next() {
			var schema, name string
			var safeName string
			err = rows.Scan(&schema, &name)
			if err != nil {
				connectionResponse.ConnectionError = fmt.Sprintf("could not read stored procedure schema: %s",err)
				return connectionResponse, nil
			}
			safeName = fmt.Sprintf(`"%s"."%s"`, schema, name)
			s.StoredProcedures = append(s.StoredProcedures, safeName)
		}
		sort.Strings(s.StoredProcedures)
	}

	s.log.Debug("Connect completed successfully.")

	return connectionResponse, err
}


func (s *Server) DiscoverSchemas(ctx context.Context, req *pub.DiscoverSchemasRequest) (*pub.DiscoverSchemasResponse, error) {
	s.log.Debug("Handling DiscoverShapesRequest...")

	if !s.connected {
		return nil, errNotConnected
	}

	var shapes []*pub.Schema
	var err error

	if req.Mode == pub.DiscoverSchemasRequest_ALL {
		if !s.settings.ShouldDisableDiscoverAll() {
			s.log.Debug("Discovering all tables and views...")
			shapes, err = s.getAllShapesFromSchema()
			s.log.Debug("Discovered tables and views.", "count", len(shapes))

			if err != nil {
				return nil, errors.Errorf("could not load tables and views from SQL: %s", err)
			}
		}
	} else {
		s.log.Debug("Refreshing schemas from request.", "count", len(req.ToRefresh))
		for _, s := range req.ToRefresh {
			shapes = append(shapes, s)
		}
	}

	resp := &pub.DiscoverSchemasResponse{}

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
				publishReq := &pub.ReadRequest{
					Schema: shape,
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
		resp.Schemas = append(resp.Schemas, shape)
	}

	sort.Sort(pub.SortableShapes(resp.Schemas))

	return resp, nil
}

func (s *Server) DiscoverShapes(ctx context.Context, req *pub.DiscoverSchemasRequest) (*pub.DiscoverSchemasResponse, error) {
	return s.DiscoverSchemas(ctx, req)
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

func (s *Server) getAllShapesFromSchema() ([]*pub.Schema, error) {

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

	var shapes []*pub.Schema

	for rows.Next() {
		shape := new(pub.Schema)

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
	NullableChar          string
	ConstraintType		  string `sql:"CONSTRAINT_TYPE"`
}

func (c columnInfo) Nullable() bool {
	return c.NullableChar == "Y"
}

func (c columnInfo) IsKey() bool {
	return c.ConstraintType == "P"
}

var deparameterizer = regexp.MustCompile(`\(\d+\)`)

func (s *Server) populateShapeColumns(shape *pub.Schema) (error) {

	var columnInfos []columnInfo

	query := shape.Query
	if query == "" {
		segs := strings.SplitN(shape.Id, ".", 2)
		if len(segs) != 2 {
			return errors.Errorf("ID %q did not have owner segment", shape.Id)
		}
		owner, table := strings.Trim(segs[0], `"`), strings.Trim(segs[1], `"`)
		query = fmt.Sprintf(`SELECT 
     c.COLUMN_NAME
	 , c.DATA_TYPE
     , c.DATA_LENGTH
     , c.DATA_PRECISION
     , c.DATA_SCALE
     , c.NULLABLE
     , tc.CONSTRAINT_TYPE
FROM ALL_TABLES t
      INNER JOIN ALL_TAB_COLUMNS c ON c.OWNER = t.OWNER AND c.TABLE_NAME = t.TABLE_NAME
      LEFT OUTER JOIN all_cons_columns ccu
                      ON ccu.COLUMN_NAME = c.COLUMN_NAME AND ccu.TABLE_NAME = t.TABLE_NAME AND
                         ccu.OWNER = t.OWNER
      LEFT OUTER JOIN SYS.ALL_CONSTRAINTS tc
                      ON tc.CONSTRAINT_NAME = ccu.CONSTRAINT_NAME AND tc.OWNER = ccu.OWNER
WHERE t.OWNER = '%s' AND t.TABLE_NAME = '%s' AND (tc.CONSTRAINT_TYPE = 'P' OR tc.CONSTRAINT_TYPE IS NULL)
ORDER BY t.TABLE_NAME`, owner, table)

		rows, err := s.executeQuery(query)
		if err != nil {
			return err
		}

		for rows.Next() {
			ci := columnInfo{}
			err = rows.Scan(&ci.ColumnName, &ci.DataType, &ci.DataLength, &ci.DataPrecision, &ci.DataScale, &ci.NullableChar, &ci.ConstraintType)
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

		property.IsKey = m.IsKey()
	}

	return nil
}

func (s *Server) ReadStream(req *pub.ReadRequest, stream pub.Publisher_ReadStreamServer) error {
	jsonReq, _ := json.Marshal(req)

	s.log.Debug("Got PublishStream request.", "req", string(jsonReq))

	if !s.connected {
		return errNotConnected
	}

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

	return err
}

func (s *Server) PublishStream(req *pub.ReadRequest, stream pub.Publisher_PublishStreamServer) error {
	return s.ReadStream(req, stream)
}

// ConfigureWrite
func (s *Server) ConfigureWrite(ctx context.Context, req *pub.ConfigureWriteRequest) (*pub.ConfigureWriteResponse, error) {
	var errArray []string

	storedProcedures, _ := json.Marshal(s.StoredProcedures)
	schemaJSON := fmt.Sprintf(`{
  "type": "object",
  "properties": {
    "storedProcedure": {
      "type": "string",
      "title": "Stored Procedure Name",
      "description": "The name of the stored procedure",
      "enum": %s
    }
  },
  "required": [
    "storedProcedure"
  ],
  "dependencies": {
    "storedProcedure": {
      "oneOf": [
        {
          "properties": {
            "storedProcedure": {
              "enum": [
                "%s"
              ]
            },
			"customName":{
			  "type": "string",
			  "title": "Custom Stored Procedure Name"
			},
			"customFullName":{
			  "type": "string",
			  "title": "Fully Qualified Custom Stored Procedure Name"
			},
            "customParameters": {
              "type": "array",
              "title": "Parameters",
              "description": "Parameters for a custom defined stored procedure",
              "items": {
                "type": "object",
                "properties": {
                  "paramName": {
                    "type": "string",
                    "title": "Parameter Name"
                  },
                  "paramType": {
                    "type": "string",
                    "title": "Parameter Type"
                  }
                }
              }
            }
          }
        }
      ]
    }
  }
}`, storedProcedures, Custom)

	// first request return ui json schema form
	if req.Form == nil || req.Form.DataJson == "" {
		return &pub.ConfigureWriteResponse{
			Form: &pub.ConfigurationFormResponse{
				DataJson:       `{"storedProcedure":""}`,
				DataErrorsJson: "",
				Errors:         nil,
				SchemaJson: schemaJSON ,
				UiJson: "",
				StateJson: "",
			},
			Schema: nil,
		}, nil
	}

	// build schema
	var query string
	var properties []*pub.Property
	var stmt *sql.Stmt
	var rows *sql.Rows
	var sprocSchema, sprocName, sprocPkg string
	var err error
	found := false
	var schemaId string
	var schemaParams strings.Builder
	var schemaProc strings.Builder
	var schemaProcOut string
	var schemaParamsOut string

	// get form data
	var formData ConfigureWriteFormData
	if err := json.Unmarshal([]byte(req.Form.DataJson), &formData); err != nil {
		errArray = append(errArray, fmt.Sprintf("error reading form data: %s", err))
		goto Done
	}

	if formData.StoredProcedure == "" {
		errArray = append(errArray, "stored procedure does not exist")
		goto Done
	}

	for _, safeProc := range s.StoredProcedures {
		if safeProc == formData.StoredProcedure {
			found = true
			s.log.Info("found stored procedure", "stored procedure", safeProc)
			continue
		}
	}

	if !found && formData.StoredProcedure != Custom {
		errArray = append(errArray, "stored procedure does not exist")
		goto Done
	}

	schemaId = formData.StoredProcedure
	if formData.StoredProcedure == Custom {
		schemaId = formData.CustomName
		s.log.Info("found custom stored procedure", "stored procedure", schemaId)
	}

	sprocSchema, sprocName = decomposeSafeName(schemaId)

	if formData.CustomFullName != "" {
		schemaId = formData.CustomFullName
	}
	schemaProc.WriteString(fmt.Sprintf("%s(", schemaId))

	s.log.Info("got decomposed schema name", "owner", sprocSchema, "object name", sprocName)

	// get params for stored procedure
	if formData.CustomFullName != "" {
		sprocSchema, sprocPkg, sprocName = decomposeSafePackageName(formData.CustomFullName)

		query = `SELECT ARGUMENT_NAME, DATA_TYPE, DATA_LENGTH, SEQUENCE FROM ALL_ARGUMENTS WHERE OWNER = :owner and OBJECT_NAME = :name and PACKAGE_NAME = :pkg order by SEQUENCE ASC`
		stmt, err = s.db.Prepare(query)
		if err != nil {
			s.log.Error(fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			errArray = append(errArray, fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			goto CustomProperties
		}

		s.log.Info("prepared query", "query", query)

		rows, err = stmt.Query(sql.Named("owner", sprocSchema), sql.Named("name", sprocName), sql.Named("pkg", sprocPkg))
		if err != nil {
			s.log.Error(fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			errArray = append(errArray, fmt.Sprintf("error getting parameters for stored procedure: %s", err))
			goto CustomProperties
		}

		s.log.Info("got rows for query", "query", query)

	} else {
		query = `SELECT ARGUMENT_NAME, DATA_TYPE, DATA_LENGTH, SEQUENCE FROM ALL_ARGUMENTS WHERE OWNER = :owner and OBJECT_NAME = :name order by SEQUENCE ASC`
		stmt, err = s.db.Prepare(query)
		if err != nil {
			s.log.Error(fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			errArray = append(errArray, fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			goto CustomProperties
		}

		s.log.Info("prepared query", "query", query)

		rows, err = stmt.Query(sql.Named("owner", sprocSchema), sql.Named("name", sprocName))
		if err != nil {
			s.log.Error(fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			errArray = append(errArray, fmt.Sprintf("error getting parameters for stored procedure: %s", err))
			goto CustomProperties
		}

		s.log.Info("got rows for query", "query", query)
	}


	// add all params to properties of schema
	for rows.Next() {
		var colName, colType string
		var length interface{}
		var sequence interface{}

		err := rows.Scan(&colName, &colType, &length, &sequence)
		if err != nil {
			s.log.Error(fmt.Sprintf("error preparing to get parameters for stored procedure: %s", err))
			errArray = append(errArray, fmt.Sprintf("error getting parameters for stored procedure: %s", err))
			goto Done
		}

		properties = append(properties, &pub.Property{
			Id: colName,
			Name: colName,
			TypeAtSource: colType,
			Type: convertFromSQLType(colType, 0),
		})

		schemaParams.WriteString(fmt.Sprintf("%s %s", colName, colType))
		if length != nil {
			schemaParams.WriteString(fmt.Sprintf("(%s)", length))
		}
		if length == nil && strings.Contains(colType, "VARCHAR2") {
			schemaParams.WriteString("(32767)")
		}
		schemaParams.WriteString(";")
		schemaProc.WriteString(fmt.Sprintf(":%s,", colName))
	}

	CustomProperties:
	if len(properties) == 0 {
		// attempt to apply user defined parameters if query does not work
		if len(formData.CustomParameters) > 0 {
			s.log.Info("building schema from custom parameters")
			for _, param := range formData.CustomParameters {
				properties = append(properties, &pub.Property{
					Id:           param.ParamName,
					Name:         param.ParamType,
					TypeAtSource: param.ParamType,
					Type:         convertFromSQLType(param.ParamType, 0),
				})

				schemaParams.WriteString(fmt.Sprintf("%s %s", param.ParamName, param.ParamType))
				schemaParams.WriteString(";")
				schemaProc.WriteString(fmt.Sprintf("%s=>:%s,", param.ParamName, param.ParamName))
			}
		}
	}

	Done:
	schemaParamsOut = schemaParams.String()
	schemaProcOut = fmt.Sprintf("%s);", strings.TrimSuffix(schemaProc.String(), ","))

	// return write back schema
	return &pub.ConfigureWriteResponse{
		Form: &pub.ConfigurationFormResponse{
			DataJson:  req.Form.DataJson,
			Errors:    errArray,
			StateJson: req.Form.StateJson,
			SchemaJson:schemaJSON,
		},
		Schema: &pub.Schema{
			Id:         schemaId,
			Query:      fmt.Sprintf("DECLARE %s BEGIN %s END;", schemaParamsOut, schemaProcOut),
			DataFlowDirection: pub.Schema_WRITE,
			Properties: properties,
		},
	}, nil
}

type ConfigureWriteFormData struct {
	StoredProcedure string `json:"storedProcedure,omitempty"`
	CustomName string `json:"customName,omitempty"`
	CustomFullName string `json:"customFullName,omitempty"`
	CustomParameters []Parameter `json:"customParameters,omitempty"`
}

type Parameter struct {
	ParamName string `json:"paramName"`
	ParamType string `json:"paramType"`
}

// PrepareWrite sets up the plugin to be able to write back
func (s *Server) PrepareWrite(ctx context.Context, req *pub.PrepareWriteRequest) (*pub.PrepareWriteResponse, error) {
	s.WriteSettings = &WriteSettings{
		Schema:    req.Schema,
		CommitSLA: req.CommitSlaSeconds,
	}

	return &pub.PrepareWriteResponse{}, nil
}

// WriteStream writes a stream of records back to the source system
func (s *Server) WriteStream(stream pub.Publisher_WriteStreamServer) error {
	// get and process each record
	for {
		// return if not configured
		if s.WriteSettings == nil {
			return nil
		}

		// get record and exit if no more records or error
		record, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var recordData map[string]interface{}
		if err := json.Unmarshal([]byte(record.DataJson), &recordData); err != nil {
			return errors.WithStack(err)
		}

		// process record and send ack
		ackMsgCh := make(chan string, 1)
		go func() {
			defer close(ackMsgCh)

			schema := s.WriteSettings.Schema

			// build params for stored procedure
			var args []interface{}
			for _, prop := range schema.Properties {

				rawValue := recordData[prop.Id]
				var value interface{}
				switch			 prop.Type {
				case pub.PropertyType_DATE, pub.PropertyType_DATETIME:
					stringValue, ok := rawValue.(string)
					if !ok {
						ackMsgCh <-fmt.Sprintf("cannot convert value %v to %s (was %T)", rawValue, prop.Type, rawValue)
						return
					}
					value, err = time.Parse(time.RFC3339, stringValue)
				default:
					value = rawValue
				}

				args = append(args, sql.Named(prop.Id, value))
			}

			// call stored procedure and capture any error
			_, err := s.db.ExecContext(context.Background(),schema.Query, args...)
			if err != nil {
				ackMsgCh <- fmt.Sprintf("could not write back: %s", err)
			}
		}()

		// send ack when done writing or on timeout
		select {
		// done writing
		case sendErr := <-ackMsgCh:
			err := stream.Send(&pub.RecordAck{
				CorrelationId: record.CorrelationId,
				Error:         sendErr,
			})
			if err != nil {
				return err
			}
		// timeout
		case <-time.After(time.Duration(s.WriteSettings.CommitSLA) * time.Second):
			err := stream.Send(&pub.RecordAck{
				CorrelationId: record.CorrelationId,
				Error:         "timed out",
			})
			if err != nil {
				return err
			}
		}
	}
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

func (s *Server) getCount(shape *pub.Schema) (*pub.Count, error) {

	cErr := make(chan error)
	cCount := make(chan int)

	go func() {
		defer close(cErr)
		defer close(cCount)

		query, err := buildQuery(&pub.ReadRequest{
			Schema: shape,
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

func (s *Server) readRecords(ctx context.Context, req *pub.ReadRequest, out chan<- *pub.Record) error {

	defer close(out)

	var err error
	var query string

	query, err = buildQuery(req)
	if err != nil {
		return errors.Errorf("could not build query: %v", err)
	}

	if req.Limit > 0 {
		query = fmt.Sprintf(`SELECT SRC.* FROM (
%s
) SRC 
WHERE rownum <= %d `, query, req.Limit)
	}

	rows, err := s.executeQuery(query)
	if err != nil {
		return errors.Errorf("error executing query %q: %v", query, err)
	}

	properties := req.Schema.Properties
	valueBuffer := make([]interface{}, len(properties))
	mapBuffer := make(map[string]interface{}, len(properties))

	for rows.Next() {
		if ctx.Err() != nil || !s.connected {
			return nil
		}

		for i := range properties {
			valueBuffer[i] = &valueBuffer[i]
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

func buildQuery(req *pub.ReadRequest) (string, error) {

	q := req.Schema.Query

	if q == "" {
		w := new(strings.Builder)
		w.WriteString("SELECT ")

		var selectors []string
		for _, p := range req.Schema.Properties {

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
		fmt.Fprintln(w, "FROM ", req.Schema.Id)

		if len(req.Filters) > 0 {
			fmt.Fprintln(w, "WHERE")

			properties := make(map[string]*pub.Property, len(req.Schema.Properties))
			for _, p := range req.Schema.Properties {
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

func decomposeSafeName(safeName string) (schema, name string) {
	segs := strings.Split(safeName, ".")
	switch len(segs) {
	case 0:
		return "", ""
	case 1:
		return "", strings.Trim(segs[0], `""`)
	case 2:
		return strings.Trim(segs[0], `""`), strings.Trim(segs[1], `""`)
	default:
		return "", ""
	}
}

func decomposeSafePackageName(safeName string) (schema, pkg, name string) {
	segs := strings.Split(safeName, ".")
	switch len(segs) {
	case 3:
		return strings.Trim(segs[0], `""`), strings.Trim(segs[1], `""`), strings.Trim(segs[2], `""`)
	default:
		return "", "", ""
	}
}

func removeSafeName(safeName string) (name string){
	segs := strings.Split(safeName, ".")
	switch len(segs) {
	case 0:
		return ""
	case 1:
		return strings.Trim(segs[0], `""`)
	case 2:
		return fmt.Sprintf("%s.%s", strings.Trim(segs[0], `""`), strings.Trim(segs[1], `""`))
	default:
		return ""
	}
}

func convertFromSQLType(t string, maxLength int) pub.PropertyType {
	text := strings.ToLower(strings.Split(t, "(")[0])

	if strings.Contains(text,"timestamp"){
		text = "timestamp"
	}

	if strings.Contains(text,"date"){
		text = "date"
	}

	switch text {
	case "timestamp":
		return pub.PropertyType_DATETIME
	case "":
		return pub.PropertyType_DATE
	case "time":
		return pub.PropertyType_TIME
	case "int", "integer", "smallint":
		return pub.PropertyType_INTEGER
	case "number", "numeric", "decimal", "dec":
		return pub.PropertyType_DECIMAL
	case "float", "binary_float", "double", "binary_double", "real":
		return pub.PropertyType_FLOAT
	case "bit":
		return pub.PropertyType_BOOL
	case "blob", "bfile", "clob", "nclob":
		return pub.PropertyType_BLOB
	case "xmltype":
		return pub.PropertyType_XML
	case "char", "varchar", "varchar2", "nchar", "nvarchar", "nvarchar2":
		if maxLength == -1 || maxLength >= 1024 {
			return pub.PropertyType_TEXT
		}
		return pub.PropertyType_STRING
	default:
		return pub.PropertyType_STRING
	}
}
