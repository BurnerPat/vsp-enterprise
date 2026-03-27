# Method-Level Source Operations

**Date:** 2026-01-06
**Report ID:** 001
**Subject:** Method-level isolation for GetSource, EditSource, and WriteSource
**Version:** v2.21.0

---

## Summary

This release introduces **method-level source operations** for ABAP classes, allowing AI assistants and developers to work with individual methods instead of entire class sources. This dramatically reduces token usage and improves precision when editing large classes.

## The Problem

When working with ABAP classes, the traditional workflow required:
1. **GetSource** → Returns entire class (often 500-2000+ lines)
2. **EditSource** → Find/replace in entire source (risk of unintended matches)
3. **WriteSource** → Replace entire class source

For a class like `ZCL_VSP_DEBUG_SERVICE` with 1035 lines and 22 methods, even a simple one-line fix required processing the entire file.

## The Solution

All three source operations now support a `method` parameter for CLAS objects:

### GetSource with method
```
GetSource(object_type="CLAS", name="ZCL_TEST", method="HANDLE_REQUEST")
```
Returns only the `METHOD...ENDMETHOD` block (~50 lines vs 1000+ lines).

### EditSource with method
```
EditSource(object_url="/sap/bc/adt/oo/classes/zcl_test",
           old_string="rv_result = 0.",
           new_string="rv_result = 42.",
           method="CALCULATE")
```
Constrains find/replace to the specified method only - prevents accidental edits in other methods with similar code.

### WriteSource with method (NEW)
```
WriteSource(object_type="CLAS", name="ZCL_TEST",
            source="  METHOD calculate.\n    rv_result = 42.\n  ENDMETHOD.",
            method="CALCULATE")
```
Replaces only the specified method implementation. The method **must already exist** in the class.

## Why This Matters

### 1. Token Efficiency
| Operation | Before | After | Savings |
|-----------|--------|-------|---------|
| Read method | 1035 lines | 50 lines | **95%** |
| Edit method | 1035 lines context | 50 lines context | **95%** |
| Write method | Full class | Method only | **95%** |

### 2. Precision
- **EditSource**: No more "matches 5 locations" errors when the same pattern exists in multiple methods
- **WriteSource**: Cannot accidentally corrupt other methods or class structure

### 3. Safety
- Method must exist - prevents creating methods in wrong places
- Syntax check before save - catches errors early
- Full activation after save - ensures class integrity

## Implementation Details

### Architecture
```
GetSource(CLAS, method=X)
    └── GetClassMethods(className)      # Get method boundaries from ADT
        └── ParseClassObjectStructure    # Parse XML with line numbers
    └── GetClassSource(className)        # Get full source
    └── Extract lines [start:end]        # Return method only

WriteSource(CLAS, method=X, source)
    └── GetClassMethods(className)       # Get method boundaries
    └── GetClassSource(className)        # Get current source
    └── Replace lines [start:end]        # Insert new method source
    └── SyntaxCheck(fullSource)          # Validate entire class
    └── UpdateSource + Activate          # Save and activate
```

### Files Changed
| File | Changes |
|------|---------|
| `pkg/adt/workflows.go` | Added `Method` to WriteSourceOptions/Result, `writeClassMethodUpdate()` |
| `internal/mcp/handlers_codeintel.go` | Added `method` parameter to WriteSource tool |

### Test Results
```
=== WriteSource with method ===
{
  "success": true,
  "objectType": "CLAS",
  "objectName": "ZCL_VSP_DEBUG_SERVICE",
  "method": "ESCAPE_JSON",
  "message": "Method ESCAPE_JSON updated and class ZCL_VSP_DEBUG_SERVICE activated successfully"
}
```

## Use Cases

### AI-Assisted Development
- **Focused context**: AI only sees the method being worked on
- **Precise edits**: No risk of hallucinating changes to other methods
- **Faster iterations**: Less data to process = faster responses

### Code Review Workflows
- Extract single method for review: `GetSource(CLAS, name, method="PROCESS")`
- Compare method implementations across classes

### Refactoring
- Safely modify one method at a time
- Automatic syntax validation before save
- Full class activation ensures integrity

## Limitations

1. **Existing methods only**: WriteSource with `method` cannot create new methods
2. **Implementation only**: Changes the METHOD...ENDMETHOD block, not the definition
3. **Single method**: Cannot update multiple methods in one call

## Future Enhancements

- Method creation support (add to definition + implementation)
- Multi-method batch updates
- Method rename with all callers update
- Method move between classes

---

## Conclusion

Method-level source operations represent a significant improvement in AI-ABAP collaboration efficiency. By reducing context size by 95%, we enable faster, more precise, and safer code modifications.

This feature is immediately available in v2.21.0 for all three source operations: GetSource, EditSource, and WriteSource.
