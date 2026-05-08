package adt

import (
	"testing"
)

func TestParseInactiveObjects(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<ioc:inactiveObjects xmlns:ioc="http://www.sap.com/adt/activation/inactiveobjects"
    xmlns:adtcore="http://www.sap.com/adt/core">
  <ioc:entry>
    <ioc:object ioc:user="DEVELOPER" ioc:deleted="false">
      <ioc:ref adtcore:uri="/sap/bc/adt/oo/classes/ZCL_TEST"
               adtcore:type="CLAS/OC"
               adtcore:name="ZCL_TEST"
               adtcore:parentUri="/sap/bc/adt/packages/$TMP"/>
    </ioc:object>
  </ioc:entry>
  <ioc:entry>
    <ioc:object ioc:user="DEVELOPER" ioc:deleted="true">
      <ioc:ref adtcore:uri="/sap/bc/adt/programs/programs/ZTEST"
               adtcore:type="PROG/P"
               adtcore:name="ZTEST"/>
    </ioc:object>
    <ioc:connection ioc:user="TRANSPORT_USER" ioc:deleted="false">
      <ioc:ref adtcore:uri="/sap/bc/adt/cts/transportrequests/DEVK900001"
               adtcore:type="TASK"
               adtcore:name="DEVK900001"/>
    </ioc:connection>
  </ioc:entry>
</ioc:inactiveObjects>`

	result, err := parseInactiveObjects([]byte(xmlData))
	if err != nil {
		t.Fatalf("parseInactiveObjects failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	// Check first entry (class without connection)
	entry1 := result[0]
	if entry1.Object == nil {
		t.Fatal("expected first entry to have object")
	}
	if entry1.Object.Name != "ZCL_TEST" {
		t.Errorf("expected name 'ZCL_TEST', got '%s'", entry1.Object.Name)
	}
	if entry1.Object.Type != "class" {
		t.Errorf("expected type 'class', got '%s'", entry1.Object.Type)
	}
	if entry1.Object.User != "DEVELOPER" {
		t.Errorf("expected user 'DEVELOPER', got '%s'", entry1.Object.User)
	}
	if entry1.Object.Deleted {
		t.Error("expected first object not to be deleted")
	}
	if entry1.Transport != nil {
		t.Error("expected first entry to have no connection")
	}

	// Check second entry (program with connection, marked deleted)
	entry2 := result[1]
	if entry2.Object == nil {
		t.Fatal("expected second entry to have object")
	}
	if entry2.Object.Name != "ZTEST" {
		t.Errorf("expected name 'ZTEST', got '%s'", entry2.Object.Name)
	}
	if !entry2.Object.Deleted {
		t.Error("expected second object to be deleted")
	}
	if entry2.Transport == nil {
		t.Fatal("expected second entry to have connection")
	}
	if entry2.Transport.Name != "DEVK900001" {
		t.Errorf("expected connection name 'DEVK900001', got '%s'", entry2.Transport.Name)
	}
}

func TestParseInactiveObjectsEmpty(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<ioc:inactiveObjects xmlns:ioc="http://www.sap.com/adt/activation/inactiveobjects">
</ioc:inactiveObjects>`

	result, err := parseInactiveObjects([]byte(xmlData))
	if err != nil {
		t.Fatalf("parseInactiveObjects failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestParseInactiveObjectsEmptyResponse(t *testing.T) {
	result, err := parseInactiveObjects([]byte{})
	if err != nil {
		t.Fatalf("parseInactiveObjects failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}
