# Rotatelog Hooks for Logrus <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

## Использование

```go

func main() {
  log := logrus.New()
  hook, err := rlog.NewHook("./access_log.%Y%m%d",
    //rotatelog.WithLinkName("./access_log"),
    rotatelog.WithMaxAge(24 * time.Hour),
    rotatelog.WithRotationTime(time.Hour),
    rotatelog.WithClock(rotatelog.UTC))

  if err != nil {
    log.Hooks.Add(hook)
  }
}
```

# Описание
Пакет собран на основе хука для logrus из репозитория https://github.com/jefurry/logrus.
Цель пакета - файловый логер для длительно запущенных приложений с поддержкой ротации лог-файлов на основании времени, размера, количества по гибко настраиваемым правилам.
Основательно переработана логика ротации по времени, добавлен коллбэк для реагирования на факт ротации файла. Добавлена опция ротации по размеру файла.

Опции
====
## Шаблон имени файла (обязательно)

Шаблон, используемый для создания фактических имен файлов журналов. Вы должны использовать шаблоны
используя формат strftime (3). Например:

```go
  rotatelog.New("/var/log/myapp/log.%Y%m%d")
```
Подсказку по форматированию можно найти здесь: https://pkg.go.dev/github.com/lestrrat-go/strftime#readme-supported-conversion-specifications

## Часы (default: rotatelog.Local)
Вы можете указать объект, реализующий интерфейс roatatelogs.Clock.
Если указана эта опция, она используется для определения текущего времени, и расчёты будут
основаны на этом. Например, если вы хотите основать свои
расчеты в формате UTC, вы можете указать Rotatelog.UTC

```go
  rotatelog.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelog.WithClock(rotatelog.UTC),
  )
```

## Symlink (по умолчанию: "")

Путь, по которому размещается symlink на фактический файл журнала. Это позволяет вам
всегда иметь файлы журналов в одном и том же месте, даже если журналы были отротированы
```go
  rotatelog.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelog.WithLinkName("/var/log/myapp/current"),
  )
```

```
  // Else where
  $ tail -f /var/log/myapp/current
```

Если не указано, symlink не будет создаваться.

## RotationTime (по умолчанию: 24 часа)

Интервал между ротацией файлов. По умолчанию журналы меняются каждые сутки.
Примечание: Не забудьте использовать значения time.Duration.

```go
  // Rotate every hour
  rotatelog.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelog.WithRotationTime(time.Hour),
  )
```
## MaxAge (по умолчанию: отключён)

Лог-файлы старше указанного порога времени будут удаляться 

Примечание. Не забудьте использовать значения time.Duration.

```go
  // Purge logs older than 1 hour
  rotatelog.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelog.WithMaxAge(time.Hour),
  )
```

## RotationCount (default: -1)

Количество файлов, по превышению которого будет происходить удаление старых лог-файлов. По умолчанию эта опция отключена.


```go
  // Purge logs except latest 7 files
  rotatelog.New(
    "/var/log/myapp/log.%Y%m%d",
    rotatelog.WithMaxAge(-1),
    rotatelog.WithRotationCount(7),
  )
```

# Ручная ротация
Если вы хотите принудительно вызвать ротацию файлов до того, как истечет фактическое время,
вы можете использовать метод Rotate(). Этот метод принудительно ротирует файлы, но
если сгенерированное имя файла конфликтует, то добавляется числовой суффикс, чтобы
новый файл не переписал существующий.

```go
rl := rotatelog.New(...)

signal.Notify(ch, syscall.SIGHUP)

go func(ch chan os.Signal) {
  <-ch
  rl.Rotate()
}()
```


