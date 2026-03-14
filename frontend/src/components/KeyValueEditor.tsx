import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, Trash2 } from 'lucide-react'

interface KeyValuePair {
  key: string
  value: string
}

interface KeyValueEditorProps {
  label: string
  pairs: KeyValuePair[]
  onChange: (pairs: KeyValuePair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
}

export function KeyValueEditor({
  label,
  pairs,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
}: KeyValueEditorProps) {
  const handleAdd = () => {
    onChange([...pairs, { key: '', value: '' }])
  }

  const handleRemove = (index: number) => {
    onChange(pairs.filter((_, i) => i !== index))
  }

  const handleChange = (index: number, field: 'key' | 'value', val: string) => {
    const updated = pairs.map((pair, i) =>
      i === index ? { ...pair, [field]: val } : pair
    )
    onChange(updated)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <label className="text-xs text-muted-foreground">{label}</label>
        <Button type="button" variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={handleAdd}>
          <Plus className="h-3 w-3 mr-1" />
          Add
        </Button>
      </div>
      {pairs.length === 0 && (
        <p className="text-xs text-muted-foreground italic">No entries</p>
      )}
      <div className="space-y-1.5">
        {pairs.map((pair, index) => (
          <div key={index} className="flex items-center gap-1.5">
            <Input
              className="h-8 text-xs"
              placeholder={keyPlaceholder}
              value={pair.key}
              onChange={(e) => handleChange(index, 'key', e.target.value)}
            />
            <Input
              className="h-8 text-xs"
              placeholder={valuePlaceholder}
              value={pair.value}
              onChange={(e) => handleChange(index, 'value', e.target.value)}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0 text-muted-foreground hover:text-unhealthy"
              onClick={() => handleRemove(index)}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}
