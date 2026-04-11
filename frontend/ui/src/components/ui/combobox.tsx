import * as React from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

interface ComboboxProps {
  value: string
  onChange: (value: string) => void
  options: string[]
  placeholder?: string
  emptyMessage?: string
  className?: string
}

/**
 * Creatable combobox — shows existing options in a dropdown and allows
 * typing a new value that isn't in the list.
 */
export function Combobox({
  value,
  onChange,
  options,
  placeholder = 'Select or type…',
  emptyMessage = 'No options found.',
  className,
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false)
  const [inputValue, setInputValue] = React.useState(value)

  // Keep local input in sync when the parent value changes (e.g. form reset)
  React.useEffect(() => {
    setInputValue(value)
  }, [value])

  function handleSelect(selected: string) {
    onChange(selected)
    setInputValue(selected)
    setOpen(false)
  }

  function handleInputChange(search: string) {
    setInputValue(search)
    // Propagate typed value immediately so the form stays in sync
    onChange(search)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn(
            'h-9 w-full justify-between px-3 font-normal',
            !value && 'text-muted-foreground',
            className
          )}
        >
          <span className="truncate">{inputValue || placeholder}</span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
        <Command>
          <CommandInput
            placeholder={placeholder}
            value={inputValue}
            onValueChange={handleInputChange}
          />
          <CommandList>
            <CommandEmpty>{emptyMessage}</CommandEmpty>
            {options.length > 0 && (
              <CommandGroup>
                {options.map((option) => (
                  <CommandItem
                    key={option}
                    value={option}
                    onSelect={() => handleSelect(option)}
                  >
                    <Check
                      className={cn(
                        'mr-2 h-4 w-4',
                        value === option ? 'opacity-100' : 'opacity-0'
                      )}
                    />
                    {option}
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
