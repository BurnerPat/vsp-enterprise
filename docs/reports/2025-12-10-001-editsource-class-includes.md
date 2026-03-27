# EditSource Class Include Support

**Date:** 2025-12-10
**Report ID:** 001
**Subject:** Full class include support for EditSource and WriteSource
**Version:** v2.12.6

---

## Summary

This release adds comprehensive support for editing class includes (testclasses, definitions, implementations, macros) via the `EditSource` tool, and fixes `WriteSource` to properly create test includes when they don't exist.

## Changes

### EditSource Class Include Support

**File:** `pkg/adt/workflows.go`

The `EditSource` function now properly handles class include URLs like:
- `/sap/bc/adt/oo/classes/ZCL_FOO/includes/testclasses`
- `/sap/bc/adt/oo/classes/ZCL_FOO/includes/definitions`
- `/sap/bc/adt/oo/classes/ZCL_FOO/includes/implementations`
- `/sap/bc/adt/oo/classes/ZCL_FOO/includes/macros`

Changes:
1. **URL Detection**: Detects class include URLs by checking for `/includes/` in the path
2. **Source Reading**: Reads source directly from include URL (no `/source/main` suffix)
3. **Locking**: Locks the parent class, not the include URL
4. **Updating**: Uses `UpdateClassInclude()` instead of `UpdateSource()`
5. **Activation**: Activates the parent class after updating

### SyntaxCheck Class Include Support

**File:** `pkg/adt/devtools.go`

The `SyntaxCheck` function now handles class include URLs properly:
- For regular objects: appends `/source/main` as before
- For class includes: uses the URL as-is (no suffix)

### WriteSource Test Include Auto-Creation

**File:** `pkg/adt/workflows.go`

When using `WriteSource` with `test_source` parameter:
1. Tries to update the test include first
2. If update fails (include doesn't exist), creates it with `CreateTestInclude()`
3. Retries the update after creation
4. Activates the test include

## Test Results

### Main Class Syntax Errors
```json
{
  "success": false,
  "syntaxErrors": [
    "Line 13: The method \"NON_EXISTENT_METHOD\" is not declared or inherited in class \"ZCL_ADT_00_GIT_DEPLOY\"."
  ],
  "message": "Edit would introduce 1 syntax errors. Changes NOT saved."
}
```

### Testclasses Syntax Errors
```json
{
  "success": false,
  "syntaxErrors": [
    "Line 11: Field \"UNDEFINED_VARIABLE\" is unknown."
  ],
  "message": "Edit would introduce 1 syntax errors. Changes NOT saved."
}
```

### Valid Testclasses Edit
```json
{
  "success": true,
  "activation": {
    "success": true,
    "messages": []
  },
  "message": "Successfully edited and activated testclasses"
}
```

### RunUnitTests After Edit
```json
{
  "classes": [
    {
      "name": "LTCL_TEST",
      "testMethods": [
        {
          "name": "TEST_DUMMY",
          "executionTime": 0
        }
      ]
    }
  ]
}
```

## Usage Examples

### Edit Test Classes
```bash
# Edit testclasses include with syntax checking
vsp EditSource \
  --object_url "/sap/bc/adt/oo/classes/zcl_my_class/includes/testclasses" \
  --old_string "assert_true( abap_true )" \
  --new_string "assert_equals( exp = 1 act = 1 )" \
  --syntax_check true
```

### Create Class with Tests
```bash
# WriteSource with test_source auto-creates test include if needed
vsp WriteSource \
  --object_type CLAS \
  --name ZCL_MY_CLASS \
  --source "CLASS zcl_my_class DEFINITION PUBLIC. ENDCLASS. CLASS zcl_my_class IMPLEMENTATION. ENDCLASS." \
  --test_source "CLASS ltcl_test DEFINITION FOR TESTING. PRIVATE SECTION. METHODS test_foo FOR TESTING. ENDCLASS. CLASS ltcl_test IMPLEMENTATION. METHOD test_foo. ENDMETHOD. ENDCLASS." \
  --package "$TMP"
```

## All Tests Pass

```
ok  github.com/oisee/vibing-steampunk/internal/mcp   0.005s
ok  github.com/oisee/vibing-steampunk/pkg/adt        0.034s
ok  github.com/oisee/vibing-steampunk/pkg/cache      0.234s
ok  github.com/oisee/vibing-steampunk/pkg/dsl        0.006s
```
