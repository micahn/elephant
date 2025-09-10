### Elephant Todo

Basic Todolist

#### Features

- basic time tracking
- create new scheduled items
- notifications for scheduled items
- mark items as: done, active
- urgent items
- clear all done items

#### Requirements

- `notify-send` for notifications

#### Usage

##### Creating a new item

By default, you can create a new item whenever no items matches the configured `min_score` threshold. If you want to, you can also configure `create_prefix`, f.e. `add`. In that case you can do `add:new item`.

If you want to create a schuduled task, you can prefix your item with either `in 5m` or `at 1500`. Possible units are `s`, `m` and `h`.

Adding a `!` suffix will mark an item as urgent.
