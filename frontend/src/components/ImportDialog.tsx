import { useState, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { api, type CheckImportResult, type CheckImportValidation } from '@/lib/api'

const EXAMPLE_YAML = `# Example: Service check configuration
# Paste your YAML here or use this template

project: my-service
environment: prod
# prune: true  # Uncomment to remove checks not in this list

defaults:
  duration: 30s
  timeout: 10s
  alert_type: slack
  alert_destination: "#my-service-alerts"

checks:
  - name: "API Health"
    type: http
    url: https://api.example.com/healthz

  - name: "TCP Connectivity"
    type: tcp
    host: api.example.com
    port: 443
    timeout: 5s

  - name: "Cron Job Monitor"
    type: passive
    timeout: 15m
    duration: 1m

  - name: "Database"
    type: pgsql_query
    host: db.example.com
    port: 5432
    pgsql:
      username: monitor
      dbname: mydb
      query: "SELECT 1"
`

interface ImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportComplete: () => void
}

type Step = 'edit' | 'preview' | 'result'

export function ImportDialog({ open, onOpenChange, onImportComplete }: ImportDialogProps) {
  const [yamlContent, setYamlContent] = useState('')
  const [step, setStep] = useState<Step>('edit')
  const [validation, setValidation] = useState<CheckImportValidation | null>(null)
  const [result, setResult] = useState<CheckImportResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reset = useCallback(() => {
    setYamlContent('')
    setStep('edit')
    setValidation(null)
    setResult(null)
    setLoading(false)
    setError(null)
  }, [])

  const handleClose = useCallback((isOpen: boolean) => {
    if (!isOpen) {
      reset()
    }
    onOpenChange(isOpen)
  }, [onOpenChange, reset])

  const handleValidate = useCallback(async () => {
    if (!yamlContent.trim()) {
      setError('Please enter YAML content')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const v = await api.validateImport(yamlContent)
      setValidation(v)
      setStep('preview')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Validation failed')
    } finally {
      setLoading(false)
    }
  }, [yamlContent])

  const handleImport = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const r = await api.importChecks(yamlContent)
      setResult(r)
      setStep('result')
      onImportComplete()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed')
    } finally {
      setLoading(false)
    }
  }, [yamlContent, onImportComplete])

  const handleLoadExample = useCallback(() => {
    setYamlContent(EXAMPLE_YAML)
  }, [])

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {step === 'edit' && 'Import Checks from YAML'}
            {step === 'preview' && 'Preview Import'}
            {step === 'result' && 'Import Results'}
          </DialogTitle>
          <DialogDescription>
            {step === 'edit' && 'Paste YAML configuration to bulk create or update checks. Supports CI-style service configs with project scoping and environment support.'}
            {step === 'preview' && 'Review the checks that will be imported.'}
            {step === 'result' && 'Import completed. See the results below.'}
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="rounded-md bg-destructive/10 border border-destructive/30 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* Step 1: YAML Editor */}
        {step === 'edit' && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <label className="text-sm text-muted-foreground">YAML Content</label>
              <Button variant="ghost" size="sm" onClick={handleLoadExample} className="text-xs">
                Load Example
              </Button>
            </div>
            <textarea
              className="w-full h-80 font-mono text-xs bg-muted/50 border border-border rounded-md p-3 resize-y focus:outline-none focus:ring-1 focus:ring-ring text-foreground"
              value={yamlContent}
              onChange={(e) => setYamlContent(e.target.value)}
              placeholder="Paste your YAML here..."
              spellCheck={false}
            />
            <div className="text-xs text-muted-foreground space-y-1">
              <p>Supported fields: <code className="bg-muted px-1 rounded">project</code>, <code className="bg-muted px-1 rounded">environment</code>, <code className="bg-muted px-1 rounded">defaults</code>, <code className="bg-muted px-1 rounded">prune</code>, <code className="bg-muted px-1 rounded">checks[]</code></p>
              <p>For CI usage: <code className="bg-muted px-1 rounded">curl -X POST -H "X-API-Key: $KEY" -H "Content-Type: application/x-yaml" --data-binary @.checker.yaml $CHECKER_URL/api/checks/import</code></p>
            </div>
          </div>
        )}

        {/* Step 2: Preview */}
        {step === 'preview' && validation && (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant={validation.valid ? 'default' : 'destructive'}>
                {validation.valid ? 'Valid' : 'Has Errors'}
              </Badge>
              <span className="text-sm text-muted-foreground">
                {validation.count} check{validation.count !== 1 ? 's' : ''} found
              </span>
            </div>

            {validation.errors && validation.errors.length > 0 && (
              <div className="rounded-md bg-destructive/10 border border-destructive/30 p-3 space-y-1">
                <p className="text-sm font-medium text-destructive">Validation Errors:</p>
                {validation.errors.map((e, i) => (
                  <p key={i} className="text-xs text-destructive">
                    Check #{e.index + 1} ({e.name || 'unnamed'}): {e.message}
                  </p>
                ))}
              </div>
            )}

            <Separator />

            <div className="rounded-lg border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50 text-muted-foreground text-xs">
                    <th className="text-left px-3 py-2 font-medium">#</th>
                    <th className="text-left px-3 py-2 font-medium">Name</th>
                    <th className="text-left px-3 py-2 font-medium">Project</th>
                    <th className="text-left px-3 py-2 font-medium">Type</th>
                    <th className="text-left px-3 py-2 font-medium">Duration</th>
                  </tr>
                </thead>
                <tbody>
                  {validation.checks.map((check, i) => (
                    <tr key={i} className="border-b border-border/50">
                      <td className="px-3 py-1.5 text-muted-foreground text-xs">{i + 1}</td>
                      <td className="px-3 py-1.5 font-medium">{String(check.name || '')}</td>
                      <td className="px-3 py-1.5 text-muted-foreground">{String(check.project || '')}</td>
                      <td className="px-3 py-1.5">
                        <Badge variant="secondary" className="text-[10px]">
                          {String(check.type || '')}
                        </Badge>
                      </td>
                      <td className="px-3 py-1.5 font-mono text-muted-foreground">{String(check.duration || '')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Step 3: Results */}
        {step === 'result' && result && (
          <div className="space-y-3">
            <div className="grid grid-cols-4 gap-3">
              <div className="rounded-md border px-3 py-2 text-center">
                <div className="text-2xl font-bold">{result.summary.created}</div>
                <div className="text-xs text-muted-foreground">Created</div>
              </div>
              <div className="rounded-md border px-3 py-2 text-center">
                <div className="text-2xl font-bold">{result.summary.updated}</div>
                <div className="text-xs text-muted-foreground">Updated</div>
              </div>
              <div className="rounded-md border px-3 py-2 text-center">
                <div className="text-2xl font-bold">{result.summary.deleted}</div>
                <div className="text-xs text-muted-foreground">Deleted</div>
              </div>
              <div className="rounded-md border px-3 py-2 text-center">
                <div className="text-2xl font-bold text-destructive">{result.summary.errors}</div>
                <div className="text-xs text-muted-foreground">Errors</div>
              </div>
            </div>

            {result.created.length > 0 && (
              <div>
                <p className="text-sm font-medium mb-1">Created:</p>
                <div className="space-y-0.5">
                  {result.created.map((item, i) => (
                    <p key={i} className="text-xs text-muted-foreground">
                      {item.project} / {item.name} <span className="font-mono">({item.uuid.slice(0, 8)})</span>
                    </p>
                  ))}
                </div>
              </div>
            )}

            {result.updated.length > 0 && (
              <div>
                <p className="text-sm font-medium mb-1">Updated:</p>
                <div className="space-y-0.5">
                  {result.updated.map((item, i) => (
                    <p key={i} className="text-xs text-muted-foreground">
                      {item.project} / {item.name} <span className="font-mono">({item.uuid.slice(0, 8)})</span>
                    </p>
                  ))}
                </div>
              </div>
            )}

            {result.deleted.length > 0 && (
              <div>
                <p className="text-sm font-medium mb-1">Deleted (pruned):</p>
                <div className="space-y-0.5">
                  {result.deleted.map((item, i) => (
                    <p key={i} className="text-xs text-muted-foreground">
                      {item.project} / {item.name} <span className="font-mono">({item.uuid.slice(0, 8)})</span>
                    </p>
                  ))}
                </div>
              </div>
            )}

            {result.errors.length > 0 && (
              <div className="rounded-md bg-destructive/10 border border-destructive/30 p-3 space-y-1">
                <p className="text-sm font-medium text-destructive">Errors:</p>
                {result.errors.map((e, i) => (
                  <p key={i} className="text-xs text-destructive">
                    {e.name || `Check #${e.index + 1}`}: {e.message}
                  </p>
                ))}
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          {step === 'edit' && (
            <>
              <Button variant="outline" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button onClick={handleValidate} disabled={loading || !yamlContent.trim()}>
                {loading ? 'Validating...' : 'Preview'}
              </Button>
            </>
          )}
          {step === 'preview' && (
            <>
              <Button variant="outline" onClick={() => setStep('edit')}>
                Back
              </Button>
              <Button
                onClick={handleImport}
                disabled={loading || !validation?.valid}
              >
                {loading ? 'Importing...' : `Import ${validation?.count || 0} Checks`}
              </Button>
            </>
          )}
          {step === 'result' && (
            <Button onClick={() => handleClose(false)}>
              Close
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
