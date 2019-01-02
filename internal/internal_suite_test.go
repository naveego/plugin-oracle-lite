package internal_test

import (
	"fmt"
	. "github.com/naveego/plugin-oracle/internal"
	"io/ioutil"
	"log"
	"os"
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
			Hostname:    "10.250.1.10",
			Port:        32769,
			ServiceName: "ORCLCDB.localdomain",
			Username:    "C##NAVEEGO",
			Password:    "temp123",
		},
	}
}

var _ = BeforeSuite(func() {
	//var err error

	os.Setenv("LD_LIBRARY_PATH", "/home/steve/src/github.com/naveego/plugin-oracle/build/oracle/linux_amd64/instantclient_18_3")
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
		_, err := db.Exec(cmd)
		Expect(err).To(Or(Not(HaveOccurred()), MatchError(ContainSubstring("table or view does not exist"))), "should execute command " + cmd)
	}

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

	err = db.Ping()
	if err != nil {
		log.Printf("Error pinging w3 database: %s", err)
		return err
	}

	return err
}

var _ = AfterSuite(func() {
	if db != nil {
		db.Close()
	}
})
