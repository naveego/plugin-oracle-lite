package internal_test

import (
	"fmt"
	. "github.com/naveego/plugin-oracle/internal"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"database/sql"
	"github.com/naveego/ci/go/build"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "gopkg.in/goracle.v2"
	"time"
)

var db *sql.DB

func TestOracle(t *testing.T) {
	RegisterFailHandler(Fail)
	build.RunSpecsWithReporting(t, "Oracle Suite")
}

func GetTestSettings() *Settings {
	return  &Settings{
		Strategy:StrategyForm,
		Form:&SettingsForm{
			Hostname:    "localhost",
			Port:        1521,
			ServiceName: "ORCLCDB",
			Username:    "C##NAVEEGO",
			Password:    "n5o_ADMIN",
		},
	}
}

var _ = BeforeSuite(func() {
	//var err error

	//os.Setenv("LD_LIBRARY_PATH", "/home/steve/src/github.com/naveego/plugin-oracle/build/oracle/linux_amd64/instantclient_18_3")
	// ldlibrarypath, ok := os.LookupEnv("LD_LIBRARY_PATH")
	// Expect(ok).To(BeTrue(), "LD_LIBRARY_PATH must be set.")
	// fmt.Println(ldlibrarypath)


	Eventually(connectToSQL, time.Second*60, time.Second).Should(Succeed())

	_, thisPath, _, _ := runtime.Caller(0)
	testDataPath := filepath.Join(thisPath, "../../test/test_data.sql")
	testDataBytes, err := ioutil.ReadFile(testDataPath)
	Expect(err).ToNot(HaveOccurred())

	cmdText := string(testDataBytes)

	cmds := strings.Split(cmdText, ";")

	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		_, err := db.Exec(cmd)
		if err != nil {
			fmt.Println(cmd)
		}

		Expect(err).To(Or(Not(HaveOccurred()), MatchError(ContainSubstring("table or view does not exist"))), "should execute command " + cmd)
	}

	cmd := `grant SELECT, INSERT, UPDATE, DELETE ON C##NAVEEGO.AGENTS to SA`
	_, err = db.Exec(cmd)
	if err != nil {
		fmt.Println(cmd)
	}
	Expect(err).To(Or(Not(HaveOccurred()), MatchError(ContainSubstring("table or view does not exist"))), "should execute command " + cmd)

	cmd = `CREATE OR REPLACE PROCEDURE "C##NAVEEGO"."TEST"(
i_AgentId IN C##NAVEEGO.AGENTS.AGENT_CODE%TYPE,
i_Name IN C##NAVEEGO.AGENTS.AGENT_NAME%TYPE,
i_Commission IN C##NAVEEGO.AGENTS.COMMISSION%TYPE)
AS
BEGIN
      UPDATE C##NAVEEGO.Agents
      SET AGENT_NAME = i_Name,
          COMMISSION = i_Commission
      WHERE AGENT_CODE = i_AgentId;
      COMMIT;
END;`
	_, err = db.Exec(cmd)
	if err != nil {
		fmt.Println(cmd)
	}
	Expect(err).To(Or(Not(HaveOccurred()), MatchError(ContainSubstring("table or view does not exist"))), "should execute command " + cmd)

	cmd = `CREATE OR REPLACE PROCEDURE SA.TEST(
i_AgentId IN "C##NAVEEGO"."AGENTS".AGENT_CODE%TYPE,
i_Name IN "C##NAVEEGO"."AGENTS".AGENT_NAME%TYPE,
i_Commission IN "C##NAVEEGO"."AGENTS".COMMISSION%TYPE)
AS
BEGIN
          "C##NAVEEGO"."TEST"(i_AgentId, i_Name, i_Commission);
END;`
	_, err = db.Exec(cmd)
	if err != nil {
		fmt.Println(cmd)
	}
	Expect(err).To(Or(Not(HaveOccurred()), MatchError(ContainSubstring("table or view does not exist"))), "should execute command " + cmd)
})

func connectToSQL() error {
	 var err error
	 var connectionString string

	connectionString, err = GetTestSettings().GetConnectionString()
	if err != nil {
		return err
	}

	fmt.Println("connectionString", connectionString)

	db, err = sql.Open("goracle", connectionString)
	if err != nil {
		log.Printf("Error connecting to SQL Host: %s", err)
		return err
	}
	err = db.Ping()
	if err != nil {
		log.Printf("Error pinging SQL Host: %s", err)
		return err
	}

	return err
}

var _ = AfterSuite(func() {
	if db != nil {
		db.Close()
	}
})
