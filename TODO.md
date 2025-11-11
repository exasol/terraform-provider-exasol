# TODO List

## High Priority

### Replace Global Delete Mutex with Retry Logic

**Status**: TODO
**Priority**: High
**Effort**: Medium

**Problem**: Currently, all delete operations are serialized using a global mutex (`internal/provider/mutex.go`) to prevent Exasol transaction collision errors (SQL error code 40001). This works but significantly slows down parallel `terraform destroy` operations.

**Current Workaround**: Global mutex lock/unlock in all Delete methods:
```go
provider.LockDelete()
defer provider.UnlockDelete()
```

**Better Solution**: Implement retry logic with exponential backoff specifically for error code 40001.

**Implementation Plan**:

1. Create a retry helper function in `internal/resources/` or `internal/provider/`:
   ```go
   // RetryOnTransactionCollision retries the operation up to maxRetries times
   // if it encounters a transaction collision error (40001)
   func RetryOnTransactionCollision(ctx context.Context, maxRetries int, operation func() error) error {
       for attempt := 0; attempt <= maxRetries; attempt++ {
           err := operation()
           if err == nil {
               return nil
           }

           // Check if this is a transaction collision (40001)
           if strings.Contains(err.Error(), "40001") && attempt < maxRetries {
               // Exponential backoff: 100ms, 200ms, 300ms
               waitTime := time.Duration(100*(attempt+1)) * time.Millisecond
               tflog.Warn(ctx, "Transaction collision detected, retrying", map[string]any{
                   "attempt": attempt + 1,
                   "maxRetries": maxRetries,
                   "waitMs": waitTime.Milliseconds(),
               })
               time.Sleep(waitTime)
               continue
           }

           return err
       }
       return fmt.Errorf("max retries exceeded")
   }
   ```

2. Update all Delete methods to use the retry helper instead of the mutex:
   ```go
   func (r *RoleGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
       // Remove: provider.LockDelete() / defer provider.UnlockDelete()

       var state roleGrantModel
       resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
       if resp.Diagnostics.HasError() {
           return
       }

       role := strings.ToUpper(state.Role.ValueString())
       grantee := strings.ToUpper(state.Grantee.ValueString())
       stmt := fmt.Sprintf(`REVOKE "%s" FROM "%s"`, role, grantee)

       // Use retry helper
       err := RetryOnTransactionCollision(ctx, 3, func() error {
           _, err := r.db.ExecContext(ctx, stmt)
           return err
       })

       if err != nil {
           resp.Diagnostics.AddError("REVOKE failed", err.Error())
       }
   }
   ```

3. Update all resource Delete methods (9 files):
   - `role_grant_resource.go`
   - `system_privilege_resource.go`
   - `object_privilege_resource.go`
   - `connection_grant_resource.go`
   - `connection_resource.go`
   - `grant_resource.go` (legacy)
   - `role_resource.go`
   - `schema_resource.go`
   - `user_resource.go`

4. Remove `internal/provider/mutex.go` and all references to `LockDelete()` / `UnlockDelete()`

5. Test with parallel destroy operations to verify collisions are handled gracefully

**Benefits**:
- ✅ Much faster parallel destroy operations
- ✅ Automatic retry on transient collision errors
- ✅ Only retries on actual collision errors, not other failures
- ✅ Better logging of retry attempts

**Testing**:
```bash
# Should complete successfully without -parallelism=1
terraform destroy -auto-approve

# Monitor logs for retry messages
TF_LOG=DEBUG terraform destroy -auto-approve 2>&1 | grep -i "transaction collision"
```

## Related Documentation

- See `CLAUDE.md` section "Important Gotchas #8" for current workaround documentation
- See `test/README.md` section "Known Issues" for user-facing documentation
