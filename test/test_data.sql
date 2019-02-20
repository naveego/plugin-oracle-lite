DROP TABLE C##NAVEEGO.Orders;
DROP TABLE C##NAVEEGO.Customers;
DROP TABLE C##NAVEEGO.Agents;
DROP TABLE C##NAVEEGO.PrePost;


DROP TABLE C##NAVEEGO.TYPES;
CREATE TABLE C##NAVEEGO.TYPES
(
  "number"                NUMBER(16,0) NOT NULL PRIMARY KEY,
  "float"                 BINARY_FLOAT,
  "double"                BINARY_DOUBLE,
  "date"                  DATE,
  "timestamp"             TIMESTAMP,
  "timestampWithTimeZone" TIMESTAMP WITH TIME ZONE,
  "intervalYear4ToMonth"  INTERVAL YEAR (2) TO MONTH,
  "intervalDay4ToSecond2" INTERVAL DAY (4) TO SECOND (2),
  "char"                  CHAR(6),
  "varchar2"              VARCHAR2(10),
  "nvarchar2"             NVARCHAR2(10),
  "nchar"                 NCHAR(6),
  "xml"                   XMLTYPE,
  "blob"                  BLOB,
  "clob"                  CLOB,
  "nclob"                 NCLOB
);


INSERT INTO C##NAVEEGO.TYPES
VALUES (42, 123456.78, -- float
        1234.5678, --DOUBLE
        DATE '1998-12-25', -- date
        TIMESTAMP '1997-01-31 09:26:56.66', -- timestamp
        timestamp '1997-01-31 09:26:56.66 +02:00', -- timestamp with timezone
        '02-04', --interval year month
        '120 6:31:14', --interval day second
        'char', -- char
        'varchar2', -- varchar
        'nvarchar2', -- nvarchar
        'nchar', -- nchar
        XMLType('<data>42</data>'), --xml
        utl_raw.cast_to_raw('blob data'),
        'clob', -- clob
        'nclobdata' --nclob
           );

CREATE TABLE C##NAVEEGO.Agents
(
  "AGENT_CODE"   CHAR(4) NOT NULL PRIMARY KEY,
  "AGENT_NAME"   VARCHAR(40),
  "WORKING_AREA" VARCHAR(35),
  "COMMISSION"   BINARY_FLOAT,
  "PHONE_NO"     CHAR(12),
  "UPDATED_AT"   timestamp with time zone,
  "BIOGRAPHY"    VARCHAR(2056)
);

INSERT INTO C##NAVEEGO.Agents
VALUES ('A007', 'Ramasundar', 'Bangalore', '0.15', '077-25814763', TIMESTAMP '1969-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A003', 'Alex', 'London', '0.13', '075-12458969', TIMESTAMP '1969-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A008', 'Alford', 'New York', '0.12', '044-25874365', TIMESTAMP '1969-01-03 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A011', 'Ravi Kumar', 'Bangalore', '0.15', '077-45625874', TIMESTAMP '1969-01-02 12:10:12', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A010', 'Santakumar', 'Chennai', '0.14', '007-22388644', TIMESTAMP '1969-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A012', 'Lucida', 'San Jose', '0.12', '044-52981425', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A005', 'Anderson', 'Brisban', '0.13', '045-21447739', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A001', 'Subbarao', 'Bangalore', '0.14', '077-12346674', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A002', 'Mukesh', 'Mumbai', '0.11', '029-12358964', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A006', 'McDen', 'London', '0.15', '078-22255588', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A004', 'Ivan', 'Torento', '0.15', '008-22544166', TIMESTAMP '1971-01-02 00:00:00', '');
INSERT INTO C##NAVEEGO.Agents
VALUES ('A009', 'Benjamin', 'Hampshair', '0.11', '008-22536178', TIMESTAMP '1971-01-02 00:00:00 -3:00', '');


CREATE TABLE C##NAVEEGO.Customers
(
  "CUST_CODE"       VARCHAR(6)    NOT NULL PRIMARY KEY,
  "CUST_NAME"       VARCHAR(40)   NOT NULL,
  "CUST_CITY"       VARCHAR(50),
  "WORKING_AREA"    VARCHAR(35)   NOT NULL,
  "CUST_COUNTRY"    VARCHAR(20)   NOT NULL,
  "GRADE"           NUMBER(6, 2),
  "OPENING_AMT"     NUMBER(12, 2) NOT NULL,
  "RECEIVE_AMT"     NUMBER(12, 2) NOT NULL,
  "PAYMENT_AMT"     NUMBER(12, 2) NOT NULL,
  "OUTSTANDING_AMT" NUMBER(12, 2) NOT NULL,
  "PHONE_NO"        VARCHAR(17)   NOT NULL,
  "AGENT_CODE"      CHAR(4)       NOT NULL REFERENCES C##NAVEEGO.Agents
);


INSERT INTO C##NAVEEGO.Customers
VALUES ('C00013',
        'Holmes',
        'London',
        'London',
        'UK',
        '2',
        '6000.00',
        '5000.00',
        '7000.00',
        '4000.00',
        'BBBBBBB',
        'A003');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00001',
        'Micheal',
        'New York',
        'New York',
        'USA',
        '2',
        '3000.00',
        '5000.00',
        '2000.00',
        '6000.00',
        'CCCCCCC',
        'A008');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00020',
        'Albert',
        'New York',
        'New York',
        'USA',
        '3',
        '5000.00',
        '7000.00',
        '6000.00',
        '6000.00',
        'BBBBSBB',
        'A008');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00025',
        'Ravindran',
        'Bangalore',
        'Bangalore',
        'India',
        '2',
        '5000.00',
        '7000.00',
        '4000.00',
        '8000.00',
        'AVAVAVA',
        'A011');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00024', 'Cook', 'London', 'London', 'UK', '2', '4000.00', '9000.00', '7000.00', '6000.00', 'FSDDSDF', 'A006');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00015',
        'Stuart',
        'London',
        'London',
        'UK',
        '1',
        '6000.00',
        '8000.00',
        '3000.00',
        '11000.00',
        'GFSGERS',
        'A003');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00002',
        'Bolt',
        'New York',
        'New York',
        'USA',
        '3',
        '5000.00',
        '7000.00',
        '9000.00',
        '3000.00',
        'DDNRDRH',
        'A008');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00018',
        'Fleming',
        'Brisban',
        'Brisban',
        'Australia',
        '2',
        '7000.00',
        '7000.00',
        '9000.00',
        '5000.00',
        'NHBGVFC',
        'A005');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00021',
        'Jacks',
        'Brisban',
        'Brisban',
        'Australia',
        '1',
        '7000.00',
        '7000.00',
        '7000.00',
        '7000.00',
        'WERTGDF',
        'A005');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00019',
        'Yearannaidu',
        'Chennai',
        'Chennai',
        'India',
        '1',
        '8000.00',
        '7000.00',
        '7000.00',
        '8000.00',
        'ZZZZBFV',
        'A010');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00005',
        'Sasikant',
        'Mumbai',
        'Mumbai',
        'India',
        '1',
        '7000.00',
        '11000.00',
        '7000.00',
        '11000.00',
        '147-25896312',
        'A002');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00007',
        'Ramanathan',
        'Chennai',
        'Chennai',
        'India',
        '1',
        '7000.00',
        '11000.00',
        '9000.00',
        '9000.00',
        'GHRDWSD',
        'A010');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00022',
        'Avinash',
        'Mumbai',
        'Mumbai',
        'India',
        '2',
        '7000.00',
        '11000.00',
        '9000.00',
        '9000.00',
        '113-12345678',
        'A002');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00004',
        'Winston',
        'Brisban',
        'Brisban',
        'Australia',
        '1',
        '5000.00',
        '8000.00',
        '7000.00',
        '6000.00',
        'AAAAAAA',
        'A005');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00023', 'Karl', 'London', 'London', 'UK', '0', '4000.00', '6000.00', '7000.00', '3000.00', 'AAAABAA', 'A006');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00006',
        'Shilton',
        'Torento',
        'Torento',
        'Canada',
        '1',
        '10000.00',
        '7000.00',
        '6000.00',
        '11000.00',
        'DDDDDDD',
        'A004');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00010',
        'Charles',
        'Hampshair',
        'Hampshair',
        'UK',
        '3',
        '6000.00',
        '4000.00',
        '5000.00',
        '5000.00',
        'MMMMMMM',
        'A009');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00017',
        'Srinivas',
        'Bangalore',
        'Bangalore',
        'India',
        '2',
        '8000.00',
        '4000.00',
        '3000.00',
        '9000.00',
        'AAAAAAB',
        'A007');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00012',
        'Steven',
        'San Jose',
        'San Jose',
        'USA',
        '1',
        '5000.00',
        '7000.00',
        '9000.00',
        '3000.00',
        'KRFYGJK',
        'A012');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00008',
        'Karolina',
        'Torento',
        'Torento',
        'Canada',
        '1',
        '7000.00',
        '7000.00',
        '9000.00',
        '5000.00',
        'HJKORED',
        'A004');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00003',
        'Martin',
        'Torento',
        'Torento',
        'Canada',
        '2',
        '8000.00',
        '7000.00',
        '7000.00',
        '8000.00',
        'MJYURFD',
        'A004');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00009',
        'Ramesh',
        'Mumbai',
        'Mumbai',
        'India',
        '3',
        '8000.00',
        '7000.00',
        '3000.00',
        '12000.00',
        'Phone No',
        'A002');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00014',
        'Rangarappa',
        'Bangalore',
        'Bangalore',
        'India',
        '2',
        '8000.00',
        '11000.00',
        '7000.00',
        '12000.00',
        'AAAATGF',
        'A001');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00016',
        'Venkatpati',
        'Bangalore',
        'Bangalore',
        'India',
        '2',
        '8000.00',
        '11000.00',
        '7000.00',
        '12000.00',
        'JRTVFDD',
        'A007');
INSERT INTO C##NAVEEGO.Customers
VALUES ('C00011',
        'Sundariya',
        'Chennai',
        'Chennai',
        'India',
        '3',
        '7000.00',
        '11000.00',
        '7000.00',
        '11000.00',
        'PPHGRTS',
        'A010');


CREATE TABLE C##NAVEEGO.Orders
(

  "ORD_NUM"         NUMBER(6, 0) NOT NULL PRIMARY KEY,
  "ORD_AMOUNT"      NUMBER(6, 2) NOT NULL,
  "ADVANCE_AMOUNT"  NUMBER(6, 2) NOT NULL,
  "ORD_DATE"        DATE         NOT NULL,
  "CUST_CODE"       VARCHAR(6)   NOT NULL,
  "AGENT_CODE"      CHAR(4)      NOT NULL,
  "ORD_DESCRIPTION" VARCHAR(60)  NOT NULL,
  CONSTRAINT fk_cust_code FOREIGN KEY (CUST_CODE) REFERENCES C##NAVEEGO.Customers (CUST_CODE),
  CONSTRAINT fk_agent_code FOREIGN KEY (AGENT_CODE) REFERENCES C##NAVEEGO.Agents (AGENT_CODE)

);


INSERT INTO C##NAVEEGO.Orders
VALUES ('200100', '1000.00', '600.00', DATE '2008-08-01', 'C00013', 'A003', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200110', '3000.00', '500.00', DATE '2008-04-15', 'C00019', 'A010', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200107', '4500.00', '900.00', DATE '2008-08-30', 'C00007', 'A010', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200112', '2000.00', '400.00', DATE '2008-05-30', 'C00016', 'A007', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200113', '4000.00', '600.00', DATE '2008-06-10', 'C00022', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200102', '2000.00', '300.00', DATE '2008-05-25', 'C00012', 'A012', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200114', '3500.00', '2000.00', DATE '2008-08-15', 'C00002', 'A008', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200122', '2500.00', '400.00', DATE '2008-09-16', 'C00003', 'A004', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200118', '500.00', '100.00', DATE '2008-07-20', 'C00023', 'A006', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200119', '4000.00', '700.00', DATE '2008-09-16', 'C00007', 'A010', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200121', '1500.00', '600.00', DATE '2008-09-23', 'C00008', 'A004', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200130', '2500.00', '400.00', DATE '2008-07-30', 'C00025', 'A011', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200134', '4200.00', '1800.00', DATE '2008-09-25', 'C00004', 'A005', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200108', '4000.00', '600.00', DATE '2008-02-15', 'C00008', 'A004', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200103', '1500.00', '700.00', DATE '2008-05-15', 'C00021', 'A005', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200105', '2500.00', '500.00', DATE '2008-07-18', 'C00025', 'A011', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200109', '3500.00', '800.00', DATE '2008-07-30', 'C00011', 'A010', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200101', '3000.00', '1000.00', DATE '2008-07-15', 'C00001', 'A008', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200111', '1000.00', '300.00', DATE '2008-07-10', 'C00020', 'A008', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200104', '1500.00', '500.00', DATE '2008-03-13', 'C00006', 'A004', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200106', '2500.00', '700.00', DATE '2008-04-20', 'C00005', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200125', '2000.00', '600.00', DATE '2008-10-10', 'C00018', 'A005', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200117', '800.00', '200.00', DATE '2008-10-20', 'C00014', 'A001', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200123', '500.00', '100.00', DATE '2008-09-16', 'C00022', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200120', '500.00', '100.00', DATE '2008-07-20', 'C00009', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200116', '500.00', '100.00', DATE '2008-07-13', 'C00010', 'A009', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200124', '500.00', '100.00', DATE '2008-06-20', 'C00017', 'A007', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200126', '500.00', '100.00', DATE '2008-06-24', 'C00022', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200129', '2500.00', '500.00', DATE '2008-07-20', 'C00024', 'A006', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200127', '2500.00', '400.00', DATE '2008-07-20', 'C00015', 'A003', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200128', '3500.00', '1500.00', DATE '2008-07-20', 'C00009', 'A002', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200135', '2000.00', '800.00', DATE '2008-09-16', 'C00007', 'A010', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200131', '900.00', '150.00', DATE '2008-08-26', 'C00012', 'A012', 'SOD');
INSERT INTO C##NAVEEGO.Orders
VALUES ('200133', '1200.00', '400.00', DATE '2008-06-29', 'C00009', 'A002', 'SOD');

CREATE TABLE C##NAVEEGO.PrePost
(
  Message varchar(50)
)