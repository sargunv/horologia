import { DatePicker, type DateValue, Portal } from "@skeletonlabs/skeleton-react";
import { Calendar, ChevronLeft, ChevronRight } from "lucide-react";

export function DateField({
  value,
  onChange,
  onOpenChange,
  disabled,
  "aria-label": ariaLabel,
}: {
  value: DateValue | null;
  onChange: (value: DateValue | null) => void;
  onOpenChange?: ((open: boolean) => void) | undefined;
  disabled?: boolean | undefined;
  "aria-label"?: string | undefined;
}) {
  return (
    <DatePicker
      value={value ? [value] : []}
      onValueChange={(e) => onChange(e.value[0] ?? null)}
      onOpenChange={(e) => onOpenChange?.(e.open)}
      closeOnSelect
      disabled={disabled}
    >
      <DatePicker.Control className="input-group preset-outlined-surface-200-800 grid grid-cols-[1fr_auto]">
        <DatePicker.Input className="ig-input" aria-label={ariaLabel} />
        <DatePicker.Trigger className="ig-btn preset-tonal-surface">
          <Calendar className="size-4" aria-hidden="true" />
        </DatePicker.Trigger>
      </DatePicker.Control>
      <Portal>
        <DatePicker.Positioner className="z-50">
          <DatePicker.Content className="card preset-outlined-surface-200-800 bg-surface-100-900 p-3">
            <DatePicker.View view="day">
              <DatePicker.ViewControl className="flex items-center justify-between gap-2">
                <DatePicker.PrevTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronLeft className="size-4" aria-hidden="true" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger className="btn btn-sm preset-tonal-surface">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronRight className="size-4" aria-hidden="true" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Context>
                {(api) => (
                  <DatePicker.Table>
                    <DatePicker.TableHead>
                      <DatePicker.TableRow>
                        {api.weekDays.map((weekDay, i) => (
                          <DatePicker.TableHeader key={i}>{weekDay.narrow}</DatePicker.TableHeader>
                        ))}
                      </DatePicker.TableRow>
                    </DatePicker.TableHead>
                    <DatePicker.TableBody>
                      {api.weeks.map((week, i) => (
                        <DatePicker.TableRow key={i}>
                          {week.map((day) => (
                            <DatePicker.TableCell key={day.toString()} value={day}>
                              <DatePicker.TableCellTrigger className="size-8 rounded-full text-sm data-[today]:font-bold data-[selected]:preset-filled-primary-500 data-[outside-range]:text-surface-400 hover:preset-tonal-surface data-[selected]:hover:preset-filled-primary-500">
                                {day.day}
                              </DatePicker.TableCellTrigger>
                            </DatePicker.TableCell>
                          ))}
                        </DatePicker.TableRow>
                      ))}
                    </DatePicker.TableBody>
                  </DatePicker.Table>
                )}
              </DatePicker.Context>
            </DatePicker.View>
            <DatePicker.View view="month">
              <DatePicker.ViewControl className="flex items-center justify-between gap-2">
                <DatePicker.PrevTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronLeft className="size-4" aria-hidden="true" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger className="btn btn-sm preset-tonal-surface">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronRight className="size-4" aria-hidden="true" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Context>
                {(api) => (
                  <DatePicker.Table>
                    <DatePicker.TableBody>
                      {api.getMonthsGrid({ columns: 4 }).map((months, i) => (
                        <DatePicker.TableRow key={i}>
                          {months.map((month) => (
                            <DatePicker.TableCell key={month.value} value={month.value} columns={4}>
                              <DatePicker.TableCellTrigger className="rounded-base px-3 py-1.5 text-sm hover:preset-tonal-surface data-[selected]:preset-filled-primary-500">
                                {month.label}
                              </DatePicker.TableCellTrigger>
                            </DatePicker.TableCell>
                          ))}
                        </DatePicker.TableRow>
                      ))}
                    </DatePicker.TableBody>
                  </DatePicker.Table>
                )}
              </DatePicker.Context>
            </DatePicker.View>
            <DatePicker.View view="year">
              <DatePicker.ViewControl className="flex items-center justify-between gap-2">
                <DatePicker.PrevTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronLeft className="size-4" aria-hidden="true" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger className="btn btn-sm preset-tonal-surface">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger className="btn-icon btn-icon-sm preset-tonal-surface">
                  <ChevronRight className="size-4" aria-hidden="true" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Context>
                {(api) => (
                  <DatePicker.Table>
                    <DatePicker.TableBody>
                      {api.getYearsGrid({ columns: 4 }).map((years, i) => (
                        <DatePicker.TableRow key={i}>
                          {years.map((year) => (
                            <DatePicker.TableCell key={year.value} value={year.value} columns={4}>
                              <DatePicker.TableCellTrigger className="rounded-base px-3 py-1.5 text-sm hover:preset-tonal-surface data-[selected]:preset-filled-primary-500">
                                {year.label}
                              </DatePicker.TableCellTrigger>
                            </DatePicker.TableCell>
                          ))}
                        </DatePicker.TableRow>
                      ))}
                    </DatePicker.TableBody>
                  </DatePicker.Table>
                )}
              </DatePicker.Context>
            </DatePicker.View>
          </DatePicker.Content>
        </DatePicker.Positioner>
      </Portal>
    </DatePicker>
  );
}
