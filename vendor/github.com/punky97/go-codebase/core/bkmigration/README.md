#BEEGRATION - MIGRATE DATABASE TOOL GUIDELINE
## Auto Migration

> Chú ý: Hiện tại chỉ hỗ trợ MySQL cho Auto Migration và môi trường Dev/Staging

**1. Cấu hình**
Cấu hình theo dạng Key/Value và đọc trực tiếp từ các file config trên remote server (Consul)

- Cấu hình thông tin MySQL
```
[mysql]
host = "localhost"
port = "3306"
user = "root"
password = "123456"
database_name = "db_test"
```

> Note: Thông tin MySQL được đọc từ Consul tại `database/sql/conf.toml`, vì vậy không cần phải khai báo nữa.
> 
- Cấu hình Auto Migration cho từng DMS
Để đảm bảo Migrate chính xác, thì từng DMS sẽ được quy định các Table được phép thực hiện Migrate
  - Auto Migration chỉ được chạy khi được Set bằng true tại `auto_migration`
  -  `include_tables`: là cột bắt buộc phải thêm cho từng DMS để biết bảng nào sẽ được áp dụng
- ON/OFF Auto Migration trên Consul:
```
[migration]
auto_migration = true
```
- Set bảng muốn thực hiện Auto Migrate tại file `migrate.json` ở trong thư mục chứa DMS Server
```
{
"include_tables" = ["user", "product"] #2 bảng user table product sẽ được chạy auto migrate.
}
```
**2. Thư mục chứa các Migrations**
Các file migrations được đọc từ path `migrations/mysql/database_name` 
ví dụ: `migrations/mysql/test_db`

## Tạo migrations

> Để tạo các Migrations thì bắt buộc bạn phải cấu hình File chứa thông tin Connect đến DB và sử dụng CLI để tạo các Migrations. Hỗ trợ viết Migrations theo Specification Language và SQL

- Cần cấu hình DB Connection ở file `db_migrations.toml` tại root của project
Ví dụ:
```
[mysql]
database_name: "database"
host: "localhost"
port: "3306"
user: "user"
password: "password"
```

- Sử dụng CLI với tên: **`beepwr`** với các lệnh như sau:
    - `migrate up table1, table2` -> Thực hiện áp dụng các Migration đang còn trạng thái Pending ở các Table 1 và Table 2 (Tương đương với thư mục 1 và 2 trong `migrations/mysql/your_database`)
    - `migrate down table1, table2` -> Thực hiện Downgrade 1 Migration tại Table 1 và Table 2
    - `migrate` -> Thực hiện áp dụng toàn bộ các Migration đang có trong thư mục Migrations (không loại trừ bất kỳ table nào)
    - `migrate status` -> Xem thông tin toàn bộ các Migration đang có

## Tạo và viết Migration

- Sử dụng CLI với tên : **`beepwr`** với các lệnh như sau:
  - `beepwr migrate add migration table_name migration_name`  
    - Ví dụ: `beepwr migrate add ezm sb_order create_table_sb_order` 
    - beepwr migrate sẽ tạo ra 2 file `up` và `down` với đuôi là `emz` và nội dung viết trong file phải là Language Specification đã được định nghĩa.
  - `beepwr migrate add sql table_name migration_name`
    - Ví dụ: `beepwr migrate add sql sb_order create_table_sb_order`
    - beepwr migrate sẽ tạo ra 2 file `up` và `down` với đuôi là `sql` và nội dung viết trong file phải là SQL scripts

## Quy tắc viết Migration

- Tên thư mục chứa tên Table phải giống như tên Table đang có trong Database
- Mỗi migration tạo ra phải có tên ý nghĩa và thực hiện nhiệm vụ cụ thể, ví dụ:
  - Tạo table product: `create_product_table` thì sẽ chỉ thực hiện tạo table product, không được sử dụng thêm các lệnh khác. Các lệnh khác thì nên để ở migration khác cho dễ maintain và versioning database ví dụ:
  `add_description_column`
- **Đã viết migration up thì phải viết migration down**

## Ví dụ viết Migrations và sử dụng

- Ta sẽ viết migrations cho bảng `product` với các migrations như sau:
- product:
**Tạo bảng product:**
  ```beepwr migrate add ezm product create_table_product```
  thư mục sẽ được tạo ra tại: `migrations/mysql/db_test/product` với 2 file là: `up.emz` và `down.emz`
  - File `up` có nội dung:
  ```
  create_table("product"){
    t.Column("id", "bigint unsigned", {primary: true})
    t.Column("name", "text", {null:false})
  }
  ```
  `t` là cú pháp bắt buộc
  - File `down` có nội dung:
  ```
  drop_table("product")
  ```
  **Thêm cột description vào bảng product**
  ```beepwr migrate add sql product add_description_column```
  thư mục product sẽ có thêm 2 file `up.sql` và `down.sql`, nội dung viết trong file này nên là SQL script
  - File up có nội dung:
  ```
  ALTER TABLE product ADD COLUMN description varchar(255)
  ```
  - File down có nội dung:
  ```
  ALTER TABLE product DROP COLUMN description
  ```

  **Sử dụng Manual Migration với lệnh:**
  ```
  beepwr migrate migrate up product
  ```

  Để thực hiện tìm và áp dụng các migrations đang có trong thư mục product cho bảng product.

  **Sử dụng Auto Migration:**
  - Cần cấu hình ở trong file config như sau:
  ```
  [migration]
  auto_migration = true
  ```
  - Cấu hình bảng trong Database sẽ được chạy Auto Migrate:
  ```
  {
  "include_tables": ["product"] #bảng product sẽ được chạy auto migrate
  }
  ```
  Auto migrations sẽ tự động chạy khi Build.

## Tài liệu viết Migrations

**- CLI Help:**

  ```
  beepwr migrate --help
  beepwr migrate add --help
  ```

**- Hướng dẫn sử dụng Specification Language để viết Migrations:**

  ```
  create_table ("sb_line_item"){
    t.Column("id", "bigint unsigned", {primary: true})
    t.Column("shop_id", "bigint unsigned")
    t.Column("order_id", "bigint unsigned")
    t.Column("product_id", "bigint unsigned")
    t.Column("variant_id", "bigint unsigned")
    t.Column("fulfillable_quantity", "int", {size:11})
    t.Column("fulfillment_service_id", "int", {size:11})
    t.Column("fulfillment_status", "int", {size:11})
    t.Column("weight", "double")
    t.Column("raw_price", "double")
    t.Column("quantity", "int")
    t.Column("requires_shipping", "boolean")
    t.Column("sku", "text")
    t.Column("title", "text")
    t.Column("variant_title", "text")
    t.Column("vendor", "text")
    t.Column("name", "text")
    t.Column("gift_card", "bool")
    t.Column("properties", "longtext")
    t.Column("taxable", "bool")
    t.Column("tax_lines", "longtext")
    t.Column("tip_payment_gateway", "text")
    t.Column("tip_payment_method", "text")
    t.Column("total_discount", "double")
    t.Column("discount_allocations", "text")
    t.Column("is_deteled", "tinyint", {size:1})
  }
  add_foreign_key("sb_line_item", "order_id", {"sb_order": ["id"]})
  ```
Trong đó `t` là cú pháp bắt buộc.

  **- Data types:**

  | **Migration type** | **MySQL**  |
  | ------------------ | ---------- |
  | string             | varchar    |
  | text               | text       |
  | integer, int       | integer    |
  | bool               | tinyint(1) |
  | time               | time       |
  | timestamp          | timestamp  |
  | datetime           | datetime   |
  | float              | decimal    |
  | double             | double     |
  | decimal            | decimal    |
  | blob               | blob       |

  **- Các options khác:**
  
  | **Migration options** | **Mô tả**                                                                                                                   |
  | --------------------- | --------------------------------------------------------------------------------------------------------------------------- |
  | primary               | Set khoá chính. Ex: {primary: true}                                                                                         |
  | size                  | Set size của Column.Ex: varchar(128) =\&gt; t.Column(&quot;column\_name&quot;, &quot;string&quot;, {&quot;size&quot;: 128}) |
  | null                  | Mặc định các column được phép null **null**.                                                                                |
  | default               | Set giá trị default cho column **null**.                                                                                    |

  **- Một số ví dụ về sử dụng các options trên:**
  - Create Table:
  ```
    create_table("table_name"){
        t.Column("id", "int", {primary: true})
        t.Column("name", "varchar", {size: 255})
    }
  ```
    
  - Drop Table:
  ```
  drop_table("foo")
  ```
  
  - Rename Table:
  ```
  rename_table("foo","bar")
  ```

  - Create Table Column:
  ```
  add_column("foo", "new_col", "string", {size:128})
  ```

  - Drop Table Column:
  ```
  drop_column("foo", "new_col")
  ```

  - Rename Table Column:
  ```
  rename_column("foo", "current_col_name", "column_name_after_rename")
  ```
  
  - Change Table Column:
  ```
  change_column("foo", "new_col", "integer")
  ```

  - Create Table Index:
  ```
  add_index("foo", "column_name", {}) #auto index name => table_name_column_name_idx
add_index("foo", "column_name", {"unique": true})
  ```

  - Drop Table Index:
  ```
  drop_index("foo", "index_name")
  ```

  - Rename Table Index:
  ```
  rename_index("foo", "old_index_name", "new_index_name")
  ```

  - Create Foreign Key:
  ```
  add_foreign_key("sb_line_item", "order_id", {"sb_order": ["id"]})

  add_foreign_key("table_name", "field", {"ref_table_name": ["ref_column"]}, {
      "name": "customize_name",
      "on_delete": "action",
      "on_update": "action",
  })
  ```

   - Drop Foreign Key:
   ```
      drop_foreign_key("table_name", "fk_name")
   ```
  
  - Raw SQL:
  ```
    sql("insert into table_name (id, name) values (1, `abc`)")
  ```