package testlib

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/store/sqlstore"
	"github.com/clear-ness/qa-discussion/store/storetest"
)

type MainHelper struct {
	Settings    *model.SqlSettings
	Store       store.Store
	SQLSupplier *sqlstore.SqlSupplier

	status int
}

func (h *MainHelper) GetStore() store.Store {
	if h.Store == nil {
		panic("MainHelper not initialized with store.")
	}

	return h.Store
}

func (h *MainHelper) GetSQLSupplier() *sqlstore.SqlSupplier {
	if h.SQLSupplier == nil {
		panic("MainHelper not initialized with sql supplier.")
	}

	return h.SQLSupplier
}

func (h *MainHelper) Main(m *testing.M) {
	h.status = m.Run()
}

// store for test
func (h *MainHelper) setupStore() {
	h.Settings = storetest.MakeSqlSettings()

	h.SQLSupplier = sqlstore.NewSqlSupplier(*h.Settings)

	h.Store = TestStore{h.SQLSupplier}
}

func (h *MainHelper) Close() error {
	if h.SQLSupplier != nil {
		h.SQLSupplier.Close()
	}

	dbStore := h.GetStore()
	// clear tables when test packages ends
	dbStore.DropAllTables()

	if r := recover(); r != nil {
		log.Fatalln(r)
	}

	os.Exit(h.status)

	return nil
}

type HelperOptions struct {
	EnableStore bool
}

func NewMainHelperWithOptions(options *HelperOptions) *MainHelper {
	var mainHelper MainHelper
	flag.Parse()

	if options != nil {
		if options.EnableStore && !testing.Short() {
			mainHelper.setupStore()
		}
	}

	return &mainHelper
}
