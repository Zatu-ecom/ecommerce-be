## **1. Category APIs**

### **Category Type System:**

```
1. GLOBAL Categories - Created by Admin, visible to all
2. SELLER Categories - Created by Seller, visible only to that seller
3. PUBLIC Categories - Global categories visible to customers
```

---

### **1.1 Create Category**

`POST /api/categories`

#### **Test Scenarios:**

| #                               | Scenario                                      | Input                                       | Expected Output                  | Priority |
| ------------------------------- | --------------------------------------------- | ------------------------------------------- | -------------------------------- | -------- |
| **Basic Creation**              |                                               |                                             |                                  |          |
| 1                               | Admin creates global root category            | name, description, type: "global"           | 201, global category created     | P0       |
| 2                               | Admin creates global subcategory              | name, parent_id (global), type: "global"    | 201, global subcategory created  | P0       |
| 3                               | Seller creates seller-specific root category  | name, description, type: "seller"           | 201, seller category created     | P0       |
| 4                               | Seller creates seller-specific subcategory    | name, parent_id (own seller cat)            | 201, seller subcategory created  | P0       |
| 5                               | Create with all optional fields               | name, desc, icon, order, is_active          | 201, category with all fields    | P1       |
| **Validation Tests**            |                                               |                                             |                                  |          |
| 6                               | Empty name validation                         | name: ""                                    | 400, VALIDATION_ERROR            | P0       |
| 7                               | Name too long                                 | name: 300 chars                             | 400, VALIDATION_ERROR            | P1       |
| 8                               | Name too short                                | name: "a"                                   | 400, VALIDATION_ERROR            | P1       |
| 9                               | Invalid characters in name                    | name: "Test<>Category"                      | 400, VALIDATION_ERROR            | P1       |
| 10                              | Whitespace-only name                          | name: " "                                   | 400, VALIDATION_ERROR            | P1       |
| 11                              | Description too long                          | description: 2000 chars                     | 400, VALIDATION_ERROR            | P2       |
| 12                              | Invalid category type                         | type: "invalid"                             | 400, VALIDATION_ERROR            | P0       |
| 13                              | Negative display order                        | order: -1                                   | 400, VALIDATION_ERROR            | P2       |
| **Duplicate Detection**         |                                               |                                             |                                  |          |
| 14                              | Duplicate global category name (admin)        | Existing global name                        | 409, CATEGORY_NAME_EXISTS        | P0       |
| 15                              | Duplicate seller category name (same seller)  | Existing seller name (same seller)          | 409, CATEGORY_NAME_EXISTS        | P0       |
| 16                              | Same name in different seller categories      | Name exists in Seller A, create in Seller B | 201, created (different scope)   | P0       |
| 17                              | Global and seller same name allowed           | Global "Electronics" + Seller "Electronics" | 201, created (different scope)   | P0       |
| **Parent Hierarchy Tests**      |                                               |                                             |                                  |          |
| 18                              | Invalid parent_id                             | Non-existent UUID                           | 404, PARENT_CATEGORY_NOT_FOUND   | P0       |
| 19                              | Seller uses global category as parent         | parent_id: global_category                  | 201, allowed                     | P1       |
| 20                              | Seller uses other seller's category as parent | parent_id: other_seller_category            | 403, FORBIDDEN                   | P0       |
| 21                              | Max nesting level (5 levels)                  | Parent at level 4                           | 400, MAX_NESTING_EXCEEDED        | P1       |
| 22                              | Parent category is inactive                   | parent_id: inactive_category                | 400, PARENT_CATEGORY_INACTIVE    | P1       |
| 23                              | Seller subcategory under global parent        | seller category -> global parent            | 201, allowed                     | P1       |
| 24                              | Global subcategory under seller parent        | global cat -> seller parent (admin)         | 400, INVALID_PARENT_TYPE         | P0       |
| **Authorization & Permissions** |                                               |                                             |                                  |          |
| 25                              | Unauthorized access (no token)                | No auth header                              | 401, UNAUTHORIZED                | P0       |
| 26                              | Customer attempts create                      | Customer token                              | 403, FORBIDDEN                   | P0       |
| 27                              | Seller creates global category                | Seller token, type: "global"                | 403, FORBIDDEN                   | P0       |
| 28                              | Admin creates seller category                 | Admin token, type: "seller"                 | 201, created (admin can)         | P1       |
| 29                              | Seller creates category for another seller    | Seller A token, seller_id: Seller B         | 403, FORBIDDEN                   | P0       |
| **Attribute Assignment**        |                                               |                                             |                                  |          |
| 30                              | Create with valid attributes                  | attribute_ids: [valid_ids]                  | 201, category with attributes    | P1       |
| 31                              | Create with invalid attribute_id              | attribute_ids: [non_existent_id]            | 404, ATTRIBUTE_NOT_FOUND         | P1       |
| 32                              | Create with empty attributes array            | attribute_ids: []                           | 201, category without attributes | P2       |
| 33                              | Create with duplicate attribute_ids           | attribute_ids: [id1, id1]                   | 400, DUPLICATE_ATTRIBUTES        | P2       |

#### **Auth Requirements:**

- ✅ Admin: Can create global and seller categories
- ✅ Seller: Can create only seller-specific categories

---

### **1.2 Get All Categories**

`GET /api/categories`

#### **Test Scenarios:**

| #                        | Scenario                                     | Query Params                                | Expected Output                            | Priority |
| ------------------------ | -------------------------------------------- | ------------------------------------------- | ------------------------------------------ | -------- |
| **Basic Retrieval**      |                                              |                                             |                                            |          |
| 1                        | Get all categories (no params) - Public      | No auth                                     | 200, only global categories                | P0       |
| 2                        | Get all categories - Customer                | Customer token                              | 200, only global categories                | P0       |
| 3                        | Get all categories - Seller                  | Seller token                                | 200, global + own seller categories        | P0       |
| 4                        | Get all categories - Admin                   | Admin token                                 | 200, all categories (global + all sellers) | P0       |
| 5                        | Get categories with hierarchy                | hierarchy=true                              | 200, nested structure                      | P0       |
| 6                        | Get flat list                                | hierarchy=false                             | 200, flat array                            | P1       |
| **Filtering by Type**    |                                              |                                             |                                            |          |
| 7                        | Filter global categories only                | type=global                                 | 200, only global categories                | P1       |
| 8                        | Filter seller categories only (seller)       | type=seller                                 | 200, only own seller categories            | P0       |
| 9                        | Filter seller categories (admin)             | type=seller&seller_id=<uuid>                | 200, specific seller categories            | P1       |
| 10                       | Customer filters seller categories           | type=seller (customer token)                | 200, empty array (no access)               | P0       |
| **Parent Filtering**     |                                              |                                             |                                            |          |
| 11                       | Get root categories only                     | parent_id=null                              | 200, only root categories                  | P1       |
| 12                       | Get subcategories of parent                  | parent_id=<uuid>                            | 200, child categories                      | P1       |
| 13                       | Get subcategories of other seller's category | parent_id=<other_seller_cat> (seller token) | 200, empty array (no access)               | P0       |
| 14                       | Get children with depth limit                | parent_id=<uuid>&depth=2                    | 200, categories up to 2 levels             | P2       |
| **Search & Filter**      |                                              |                                             |                                            |          |
| 15                       | Search by name                               | search=Electronics                          | 200, matching categories                   | P1       |
| 16                       | Search case-insensitive                      | search=electronics                          | 200, matching categories                   | P1       |
| 17                       | Search partial match                         | search=Elect                                | 200, categories starting with "Elect"      | P1       |
| 18                       | Search in seller scope                       | search=Phone (seller token)                 | 200, matches in global + own seller        | P1       |
| 19                       | Filter by active status                      | is_active=true                              | 200, only active categories                | P1       |
| 20                       | Filter by inactive status                    | is_active=false                             | 200, only inactive (admin/seller)          | P2       |
| **Pagination**           |                                              |                                             |                                            |          |
| 21                       | First page                                   | page=1&limit=10                             | 200, first 10 categories                   | P0       |
| 22                       | Second page                                  | page=2&limit=10                             | 200, next 10 categories                    | P1       |
| 23                       | Custom page size                             | page=1&limit=50                             | 200, 50 categories                         | P1       |
| 24                       | Invalid page number                          | page=0                                      | 400, VALIDATION_ERROR                      | P2       |
| 25                       | Negative page number                         | page=-1                                     | 400, VALIDATION_ERROR                      | P2       |
| 26                       | Invalid limit                                | limit=0                                     | 400, VALIDATION_ERROR                      | P2       |
| 27                       | Limit too large                              | limit=1000                                  | 400, VALIDATION_ERROR                      | P2       |
| 28                       | Empty result set                             | search=NonExistentCategory                  | 200, empty array with pagination           | P2       |
| **Additional Data**      |                                              |                                             |                                            |          |
| 29                       | Include product count                        | include_count=true                          | 200, categories with product_count         | P1       |
| 30                       | Include subcategory count                    | include_children_count=true                 | 200, categories with children_count        | P2       |
| 31                       | Include full hierarchy path                  | include_path=true                           | 200, categories with breadcrumb path       | P2       |
| 32                       | Include attributes                           | include_attributes=true                     | 200, categories with attributes            | P1       |
| **Data Isolation Tests** |                                              |                                             |                                            |          |
| 33                       | Seller A cannot see Seller B categories      | Seller A token, no filters                  | 200, only global + Seller A                | P0       |
| 34                       | Seller B cannot see Seller A categories      | Seller B token, no filters                  | 200, only global + Seller B                | P0       |
| 35                       | Admin sees all seller categories             | Admin token                                 | 200, global + all sellers                  | P0       |
| 36                       | Public/Customer sees no seller categories    | No auth or customer token                   | 200, only global                           | P0       |
| **Sorting**              |                                              |                                             |                                            |          |
| 37                       | Sort by name ascending                       | sort_by=name&order=asc                      | 200, alphabetically sorted                 | P1       |
| 38                       | Sort by name descending                      | sort_by=name&order=desc                     | 200, reverse alphabetically                | P1       |
| 39                       | Sort by display order                        | sort_by=order&order=asc                     | 200, sorted by display_order               | P1       |
| 40                       | Sort by created date                         | sort_by=created_at&order=desc               | 200, newest first                          | P2       |
| 41                       | Invalid sort field                           | sort_by=invalid                             | 400, VALIDATION_ERROR                      | P2       |
| **Combined Filters**     |                                              |                                             |                                            |          |
| 42                       | Search + type + active                       | search=Phone&type=global&is_active=true     | 200, matching results                      | P1       |
| 43                       | Hierarchy + pagination                       | hierarchy=true&page=1&limit=20              | 200, paginated tree                        | P2       |
| 44                       | Parent + search                              | parent_id=<uuid>&search=Mobile              | 200, matching children                     | P2       |

#### **Auth Requirements:**

- ✅ Public: Global categories only
- ✅ Customer: Global categories only
- ✅ Seller: Global + Own seller categories
- ✅ Admin: All categories

---

### **1.3 Get Category by ID**

`GET /api/categories/:id`

#### **Test Scenarios:**

| #                       | Scenario                                         | Input                                  | Expected Output                    | Priority |
| ----------------------- | ------------------------------------------------ | -------------------------------------- | ---------------------------------- | -------- |
| **Basic Retrieval**     |                                                  |                                        |                                    |          |
| 1                       | Get global category (public)                     | Global category ID, no auth            | 200, category details              | P0       |
| 2                       | Get global category (customer)                   | Global category ID, customer token     | 200, category details              | P0       |
| 3                       | Get global category (seller)                     | Global category ID, seller token       | 200, category details              | P0       |
| 4                       | Get own seller category (seller)                 | Own seller category ID, seller token   | 200, category details              | P0       |
| 5                       | Get other seller category (seller)               | Other seller category ID, seller token | 404, CATEGORY_NOT_FOUND            | P0       |
| 6                       | Get any seller category (admin)                  | Any seller category ID, admin token    | 200, category details              | P0       |
| **Data Isolation**      |                                                  |                                        |                                    |          |
| 7                       | Public tries to access seller category           | Seller category ID, no auth            | 404, CATEGORY_NOT_FOUND            | P0       |
| 8                       | Customer tries to access seller category         | Seller category ID, customer token     | 404, CATEGORY_NOT_FOUND            | P0       |
| 9                       | Seller A tries Seller B's category               | Seller B category ID, Seller A token   | 404, CATEGORY_NOT_FOUND            | P0       |
| 10                      | Seller accesses own inactive category            | Own inactive category, seller token    | 200, category details              | P1       |
| **Validation**          |                                                  |                                        |                                    |          |
| 11                      | Category not found                               | Non-existent UUID                      | 404, CATEGORY_NOT_FOUND            | P0       |
| 12                      | Invalid UUID format                              | "abc123"                               | 400, INVALID_UUID                  | P1       |
| 13                      | Empty UUID                                       | ""                                     | 400, INVALID_UUID                  | P2       |
| 14                      | Malformed UUID                                   | "123e4567-e89b"                        | 400, INVALID_UUID                  | P2       |
| **Include Options**     |                                                  |                                        |                                    |          |
| 15                      | Include attributes                               | include_attributes=true                | 200, category + attributes         | P1       |
| 16                      | Include products                                 | include_products=true                  | 200, category + products (limited) | P2       |
| 17                      | Include product count                            | include_product_count=true             | 200, category + product_count      | P1       |
| 18                      | Include parent details                           | include_parent=true                    | 200, category + parent info        | P1       |
| 19                      | Include children                                 | include_children=true                  | 200, category + child categories   | P1       |
| 20                      | Include full hierarchy path                      | include_path=true                      | 200, category + breadcrumb         | P1       |
| 21                      | Include all relations                            | include_all=true                       | 200, category + all relations      | P2       |
| **Status & Visibility** |                                                  |                                        |                                    |          |
| 22                      | Get inactive global category (public)            | Inactive global ID, no auth            | 404, CATEGORY_NOT_FOUND            | P1       |
| 23                      | Get inactive global category (admin)             | Inactive global ID, admin token        | 200, category details              | P1       |
| 24                      | Get inactive seller category (seller)            | Own inactive category, seller token    | 200, category details              | P1       |
| 25                      | Get deleted/soft-deleted category                | Deleted category ID                    | 404, CATEGORY_NOT_FOUND            | P1       |
| **Metadata**            |                                                  |                                        |                                    |          |
| 26                      | Response includes created_at                     | Valid ID                               | 200, has created_at timestamp      | P2       |
| 27                      | Response includes updated_at                     | Valid ID                               | 200, has updated_at timestamp      | P2       |
| 28                      | Response includes created_by (admin view)        | Valid ID, admin token                  | 200, has created_by user_id        | P2       |
| 29                      | Response includes category type                  | Valid ID                               | 200, has type (global/seller)      | P1       |
| 30                      | Response includes seller_id (if seller category) | Seller category ID                     | 200, has seller_id                 | P1       |

#### **Auth Requirements:**

- ✅ Public: Global active categories only
- ✅ Customer: Global active categories only
- ✅ Seller: Global + Own seller categories (active + inactive)
- ✅ Admin: All categories (any type, any status)

---

### **1.4 Update Category**

`PUT /api/categories/:id`

#### **Test Scenarios:**

| #                               | Scenario                                    | Input                                     | Expected Output                | Priority |
| ------------------------------- | ------------------------------------------- | ----------------------------------------- | ------------------------------ | -------- |
| **Basic Updates**               |                                             |                                           |                                |          |
| 1                               | Admin updates global category name          | New name, admin token                     | 200, updated category          | P0       |
| 2                               | Admin updates global category description   | New description, admin token              | 200, updated category          | P0       |
| 3                               | Seller updates own category name            | New name, seller token                    | 200, updated category          | P0       |
| 4                               | Seller updates own category description     | New description, seller token             | 200, updated category          | P0       |
| 5                               | Update display order                        | New order value                           | 200, updated order             | P1       |
| 6                               | Update category icon/image                  | New icon URL                              | 200, updated icon              | P2       |
| 7                               | Update is_active status                     | is_active: false                          | 200, category deactivated      | P1       |
| 8                               | Partial update (only name)                  | Only name field                           | 200, only name updated         | P1       |
| **Parent Hierarchy Changes**    |                                             |                                           |                                |          |
| 9                               | Move category to new parent (admin)         | New valid parent_id                       | 200, category moved            | P1       |
| 10                              | Move seller category to global parent       | Global parent_id, seller token            | 200, moved                     | P1       |
| 11                              | Move seller category to own seller parent   | Own seller parent_id                      | 200, moved                     | P1       |
| 12                              | Move global to seller parent (admin try)    | Seller parent_id, admin token             | 400, INVALID_PARENT_TYPE       | P0       |
| 13                              | Seller moves to other seller's parent       | Other seller parent_id, seller token      | 404, PARENT_NOT_FOUND          | P0       |
| 14                              | Move category to itself as parent           | parent_id: self                           | 400, CIRCULAR_REFERENCE        | P0       |
| 15                              | Move category to descendant as parent       | parent_id: child/grandchild               | 400, CIRCULAR_REFERENCE        | P0       |
| 16                              | Move to parent causing max depth exceed     | Parent causing depth > 5                  | 400, MAX_NESTING_EXCEEDED      | P1       |
| 17                              | Remove parent (make root)                   | parent_id: null                           | 200, category is now root      | P1       |
| **Validation Tests**            |                                             |                                           |                                |          |
| 18                              | Empty name                                  | name: ""                                  | 400, VALIDATION_ERROR          | P0       |
| 19                              | Name too long                               | name: 300 chars                           | 400, VALIDATION_ERROR          | P1       |
| 20                              | Whitespace-only name                        | name: " "                                 | 400, VALIDATION_ERROR          | P1       |
| 21                              | Invalid characters                          | name: "Test<script>"                      | 400, VALIDATION_ERROR          | P1       |
| 22                              | Description too long                        | description: 2000 chars                   | 400, VALIDATION_ERROR          | P2       |
| **Duplicate Detection**         |                                             |                                           |                                |          |
| 23                              | Duplicate global name (admin)               | Existing global name                      | 409, CATEGORY_NAME_EXISTS      | P0       |
| 24                              | Duplicate seller name (same seller)         | Existing name in same seller              | 409, CATEGORY_NAME_EXISTS      | P0       |
| 25                              | Same name as other seller category          | Name exists in Seller A, Seller B updates | 200, allowed (different scope) | P0       |
| 26                              | Update to same name (no change)             | Current name                              | 200, updated (idempotent)      | P2       |
| **Authorization & Permissions** |                                             |                                           |                                |          |
| 27                              | Unauthorized access                         | No token                                  | 401, UNAUTHORIZED              | P0       |
| 28                              | Customer attempts update                    | Customer token                            | 403, FORBIDDEN                 | P0       |
| 29                              | Seller updates global category              | Seller token, global category             | 403, FORBIDDEN                 | P0       |
| 30                              | Seller updates other seller's category      | Seller A token, Seller B category         | 404, CATEGORY_NOT_FOUND        | P0       |
| 31                              | Admin updates any global category           | Admin token, any global category          | 200, updated                   | P0       |
| 32                              | Admin updates any seller category           | Admin token, any seller category          | 200, updated                   | P0       |
| 33                              | Seller updates own inactive category        | Seller token, own inactive category       | 200, updated                   | P1       |
| **Not Found & Invalid ID**      |                                             |                                           |                                |          |
| 34                              | Category not found                          | Non-existent UUID                         | 404, CATEGORY_NOT_FOUND        | P0       |
| 35                              | Invalid UUID format                         | "abc123"                                  | 400, INVALID_UUID              | P1       |
| 36                              | Deleted category                            | Soft-deleted category ID                  | 404, CATEGORY_NOT_FOUND        | P1       |
| **Attribute Updates**           |                                             |                                           |                                |          |
| 37                              | Update category attributes                  | New attribute_ids[]                       | 200, attributes updated        | P1       |
| 38                              | Add new attributes                          | Append to attribute_ids[]                 | 200, attributes added          | P1       |
| 39                              | Remove attributes                           | Remove from attribute_ids[]               | 200, attributes removed        | P1       |
| 40                              | Invalid attribute_id                        | Non-existent attribute                    | 404, ATTRIBUTE_NOT_FOUND       | P1       |
| 41                              | Duplicate attributes in request             | attribute_ids: [id1, id1]                 | 400, DUPLICATE_ATTRIBUTES      | P2       |
| **Type Change Restrictions**    |                                             |                                           |                                |          |
| 42                              | Change category type global->seller (admin) | type: "seller"                            | 400, TYPE_CHANGE_NOT_ALLOWED   | P0       |
| 43                              | Change category type seller->global (admin) | type: "global"                            | 400, TYPE_CHANGE_NOT_ALLOWED   | P0       |
| 44                              | Seller attempts type change                 | type: "global", seller token              | 403, FORBIDDEN                 | P0       |
| **Business Rules**              |                                             |                                           |                                |          |
| 45                              | Update category with products               | Category has products                     | 200, updated (allowed)         | P1       |
| 46                              | Update category with subcategories          | Category has children                     | 200, updated (allowed)         | P1       |
| 47                              | Deactivate category with products           | is_active: false, has products            | 200, deactivated (warning)     | P1       |
| 48                              | Deactivate category with active children    | is_active: false, has active children     | 400, HAS_ACTIVE_CHILDREN       | P1       |
| **Concurrent Updates**          |                                             |                                           |                                |          |
| 49                              | Optimistic locking (if implemented)         | Update with stale version                 | 409, CONFLICT                  | P2       |
| 50                              | Concurrent same-name updates                | Two sellers updating to same name         | One succeeds, one fails 409    | P2       |

#### **Auth Requirements:**

- ✅ Admin: Can update any category (global or seller)
- ✅ Seller: Can update only own seller categories
- ❌ Customer: Cannot update categories
- ❌ Public: Cannot update categories

---

### **1.5 Delete Category**

`DELETE /api/categories/:id`

#### **Test Scenarios:**

| #                               | Scenario                                     | Input                                    | Expected Output                         | Priority |
| ------------------------------- | -------------------------------------------- | ---------------------------------------- | --------------------------------------- | -------- |
| **Basic Deletion**              |                                              |                                          |                                         |          |
| 1                               | Admin deletes empty global category          | Global category, no products/children    | 204, deleted                            | P0       |
| 2                               | Admin deletes empty seller category          | Seller category, no products/children    | 204, deleted                            | P0       |
| 3                               | Seller deletes own empty category            | Own category, no products/children       | 204, deleted                            | P0       |
| 4                               | Soft delete (default behavior)               | Delete request                           | 200, category soft-deleted              | P1       |
| 5                               | Hard delete (force)                          | force=true, admin only                   | 204, permanently deleted                | P2       |
| **Deletion Restrictions**       |                                              |                                          |                                         |          |
| 6                               | Delete category with products                | Category has products                    | 400, CATEGORY_HAS_PRODUCTS              | P0       |
| 7                               | Delete category with subcategories           | Category has children                    | 400, CATEGORY_HAS_CHILDREN              | P0       |
| 8                               | Delete category with both                    | Category has products + children         | 400, CATEGORY_HAS_PRODUCTS_AND_CHILDREN | P0       |
| 9                               | Delete category with inactive products       | Has inactive products                    | 400, CATEGORY_HAS_PRODUCTS              | P1       |
| 10                              | Delete category with inactive children       | Has inactive children                    | 400, CATEGORY_HAS_CHILDREN              | P1       |
| **Force/Cascade Options**       |                                              |                                          |                                         |          |
| 11                              | Force delete with products (admin)           | force=true, has products                 | 400, CANNOT_DELETE_WITH_PRODUCTS        | P0       |
| 12                              | Cascade delete with children (admin)         | cascade=true, has children               | 204, deleted with children              | P2       |
| 13                              | Reassign products before delete              | reassign_to=<category_id>                | 204, products moved, deleted            | P2       |
| 14                              | Reassign children before delete              | reassign_parent=<category_id>            | 204, children moved, deleted            | P2       |
| **Authorization & Permissions** |                                              |                                          |                                         |          |
| 15                              | Unauthorized access                          | No token                                 | 401, UNAUTHORIZED                       | P0       |
| 16                              | Customer attempts delete                     | Customer token                           | 403, FORBIDDEN                          | P0       |
| 17                              | Seller deletes global category               | Seller token, global category            | 403, FORBIDDEN                          | P0       |
| 18                              | Seller deletes other seller's category       | Seller A token, Seller B category        | 404, CATEGORY_NOT_FOUND                 | P0       |
| 19                              | Admin deletes any global category            | Admin token, any global category         | 204/400 based on content                | P0       |
| 20                              | Admin deletes any seller category            | Admin token, any seller category         | 204/400 based on content                | P0       |
| 21                              | Seller deletes own category with products    | Seller token, own category, has products | 400, CATEGORY_HAS_PRODUCTS              | P0       |
| **Not Found & Invalid ID**      |                                              |                                          |                                         |          |
| 22                              | Category not found                           | Non-existent UUID                        | 404, CATEGORY_NOT_FOUND                 | P0       |
| 23                              | Invalid UUID format                          | "abc123"                                 | 400, INVALID_UUID                       | P1       |
| 24                              | Already deleted category                     | Soft-deleted category ID                 | 404, CATEGORY_NOT_FOUND                 | P1       |
| 25                              | Empty ID                                     | Empty string                             | 400, INVALID_UUID                       | P2       |
| **Data Isolation**              |                                              |                                          |                                         |          |
| 26                              | Seller A cannot delete Seller B's category   | Seller A token, Seller B category        | 404, CATEGORY_NOT_FOUND                 | P0       |
| 27                              | Seller cannot see other's deleted categories | After deletion by other seller           | 404, CATEGORY_NOT_FOUND                 | P0       |
| 28                              | Public cannot delete any category            | No auth, any category                    | 401, UNAUTHORIZED                       | P0       |
| **Cascading Effects**           |                                              |                                          |                                         |          |
| 29                              | Delete updates parent's children count       | Delete child category                    | 204, parent updated                     | P2       |
| 30                              | Delete category updates product count        | Category with reassigned products        | 204, counts updated                     | P2       |
| 31                              | Orphaned children after deletion             | Children become root or reassigned       | 204, children handled                   | P2       |
| **Restore After Soft Delete**   |                                              |                                          |                                         |          |
| 32                              | Restore soft-deleted category (admin)        | POST /categories/:id/restore             | 200, category restored                  | P2       |
| 33                              | Cannot restore hard-deleted                  | Permanently deleted category             | 404, CATEGORY_NOT_FOUND                 | P2       |
| 34                              | Seller restores own soft-deleted             | Own soft-deleted category                | 200, restored                           | P2       |
| **Edge Cases**                  |                                              |                                          |                                         |          |
| 35                              | Delete root category with deep tree          | Root with 4 levels of children           | 400, CATEGORY_HAS_CHILDREN              | P1       |
| 36                              | Delete last category in seller scope         | Seller's only category                   | 204, deleted (allowed)                  | P1       |
| 37                              | Delete category in use by active orders      | Category products in orders              | 400, CATEGORY_IN_ACTIVE_ORDERS          | P1       |
| 38                              | Delete category twice (idempotent)           | Delete same category again               | 404, CATEGORY_NOT_FOUND                 | P2       |
| **Bulk Delete**                 |                                              |                                          |                                         |          |
| 39                              | Bulk delete multiple categories (admin)      | DELETE with ids[]                        | 207, multi-status response              | P2       |
| 40                              | Bulk delete with some failures               | Mix of valid/invalid IDs                 | 207, partial success                    | P2       |

#### **Auth Requirements:**

- ✅ Admin: Can delete any category (with restrictions)
- ✅ Seller: Can delete only own seller categories (with restrictions)
- ❌ Customer: Cannot delete categories
- ❌ Public: Cannot delete categories

---

### **1.6 Get Category Attributes**

`GET /api/categories/:id/attributes`

#### **Test Scenarios:**

| #                          | Scenario                                     | Input                             | Expected Output                        | Priority |
| -------------------------- | -------------------------------------------- | --------------------------------- | -------------------------------------- | -------- |
| **Basic Retrieval**        |                                              |                                   |                                        |          |
| 1                          | Get attributes of global category (public)   | Global category ID, no auth       | 200, attribute list                    | P0       |
| 2                          | Get attributes of seller category (seller)   | Own seller category, seller token | 200, attribute list                    | P0       |
| 3                          | Get attributes of seller category (public)   | Seller category, no auth          | 404, CATEGORY_NOT_FOUND                | P0       |
| 4                          | Category has no attributes                   | Category without attributes       | 200, empty array                       | P1       |
| 5                          | Get attributes with full details             | include_details=true              | 200, attributes with metadata          | P1       |
| **Inheritance**            |                                              |                                   |                                        |          |
| 6                          | Include inherited from parent                | inherit=true                      | 200, parent attributes included        | P1       |
| 7                          | Include inherited from all ancestors         | inherit=true&all_ancestors=true   | 200, full attribute hierarchy          | P1       |
| 8                          | Exclude inherited attributes                 | inherit=false (default)           | 200, only direct attributes            | P1       |
| 9                          | Inherited attributes marked as such          | inherit=true                      | 200, attributes with is_inherited flag | P2       |
| **Data Isolation**         |                                              |                                   |                                        |          |
| 10                         | Seller A gets Seller B's category attributes | Seller B category, Seller A token | 404, CATEGORY_NOT_FOUND                | P0       |
| 11                         | Public gets seller category attributes       | Seller category, no auth          | 404, CATEGORY_NOT_FOUND                | P0       |
| 12                         | Admin gets any category attributes           | Any category, admin token         | 200, attribute list                    | P0       |
| **Not Found & Validation** |                                              |                                   |                                        |          |
| 13                         | Category not found                           | Non-existent UUID                 | 404, CATEGORY_NOT_FOUND                | P0       |
| 14                         | Invalid UUID format                          | "abc123"                          | 400, INVALID_UUID                      | P1       |
| 15                         | Inactive category (public)                   | Inactive global category, no auth | 404, CATEGORY_NOT_FOUND                | P1       |
| 16                         | Inactive category (admin)                    | Inactive category, admin token    | 200, attribute list                    | P1       |
| **Filtering & Sorting**    |                                              |                                   |                                        |          |
| 17                         | Filter by attribute type                     | type=text                         | 200, only text attributes              | P2       |
| 18                         | Filter required attributes only              | required=true                     | 200, only required attributes          | P2       |
| 19                         | Sort by display order                        | sort_by=order                     | 200, sorted by order                   | P2       |
| 20                         | Sort by name                                 | sort_by=name                      | 200, alphabetically sorted             | P2       |
| **Response Details**       |                                              |                                   |                                        |          |
| 21                         | Response includes attribute metadata         | Valid category                    | 200, includes type, required, options  | P1       |
| 22                         | Response includes validation rules           | Valid category                    | 200, includes min, max, pattern        | P2       |
| 23                         | Response includes default values             | Valid category                    | 200, includes defaults                 | P2       |

#### **Auth Requirements:**

- ✅ Public: Global categories only
- ✅ Customer: Global categories only
- ✅ Seller: Global + Own seller categories
- ✅ Admin: All categories

---

### **1.7 Get Seller Categories** ⭐ NEW

`GET /api/sellers/:seller_id/categories`

#### **Test Scenarios:**

| #                   | Scenario                           | Query Params                | Expected Output                 | Priority |
| ------------------- | ---------------------------------- | --------------------------- | ------------------------------- | -------- |
| **Basic Retrieval** |                                    |                             |                                 |          |
| 1                   | Seller gets own categories         | Own seller_id, seller token | 200, own categories             | P0       |
| 2                   | Admin gets any seller's categories | Any seller_id, admin token  | 200, seller's categories        | P0       |
| 3                   | Other seller tries to access       | Seller A ID, Seller B token | 403, FORBIDDEN                  | P0       |
| 4                   | Public tries to access             | Seller ID, no auth          | 403, FORBIDDEN                  | P0       |
| 5                   | Customer tries to access           | Seller ID, customer token   | 403, FORBIDDEN                  | P0       |
| **Filtering**       |                                    |                             |                                 |          |
| 6                   | Filter by active status            | is_active=true              | 200, only active categories     | P1       |
| 7                   | Filter root categories only        | parent_id=null              | 200, root categories            | P1       |
| 8                   | Search by name                     | search=Electronics          | 200, matching categories        | P1       |
| 9                   | Include global categories          | include_global=true         | 200, global + seller categories | P1       |
| **Validation**      |                                    |                             |                                 |          |
| 10                  | Invalid seller_id                  | Non-existent seller UUID    | 404, SELLER_NOT_FOUND           | P0       |
| 11                  | Seller not found                   | Non-existent UUID           | 404, SELLER_NOT_FOUND           | P0       |
| 12                  | Seller has no categories           | Valid seller, no categories | 200, empty array                | P1       |

#### **Auth Requirements:**

- ✅ Owner Seller: Own categories only
- ✅ Admin: Any seller's categories
- ❌ Other Sellers: Cannot access
- ❌ Customer/Public: Cannot access

---

### **1.8 Assign Attributes to Category** ⭐ NEW

`POST /api/categories/:id/attributes`

#### **Test Scenarios:**

| #                    | Scenario                                    | Input                               | Expected Output             | Priority |
| -------------------- | ------------------------------------------- | ----------------------------------- | --------------------------- | -------- |
| **Basic Assignment** |                                             |                                     |                             |          |
| 1                    | Admin assigns attributes to global category | attribute_ids[], admin token        | 200, attributes assigned    | P0       |
| 2                    | Seller assigns attributes to own category   | attribute_ids[], seller token       | 200, attributes assigned    | P0       |
| 3                    | Assign single attribute                     | attribute_ids: [id1]                | 200, attribute assigned     | P0       |
| 4                    | Assign multiple attributes                  | attribute_ids: [id1, id2, id3]      | 200, all assigned           | P0       |
| 5                    | Assign with display order                   | attributes with order values        | 200, assigned with order    | P1       |
| 6                    | Assign with required flag                   | attributes with is_required flag    | 200, assigned as required   | P1       |
| **Validation**       |                                             |                                     |                             |          |
| 7                    | Invalid attribute_id                        | Non-existent attribute              | 404, ATTRIBUTE_NOT_FOUND    | P0       |
| 8                    | Empty attribute array                       | attribute_ids: []                   | 400, EMPTY_ATTRIBUTE_LIST   | P1       |
| 9                    | Duplicate attributes                        | attribute_ids: [id1, id1]           | 400, DUPLICATE_ATTRIBUTES   | P1       |
| 10                   | Attribute already assigned                  | Re-assign same attribute            | 200, idempotent (no change) | P2       |
| **Authorization**    |                                             |                                     |                             |          |
| 11                   | Seller assigns to global category           | Global category, seller token       | 403, FORBIDDEN              | P0       |
| 12                   | Seller assigns to other's category          | Other seller category, seller token | 404, CATEGORY_NOT_FOUND     | P0       |
| 13                   | Unauthorized access                         | No token                            | 401, UNAUTHORIZED           | P0       |
| 14                   | Customer attempts assignment                | Customer token                      | 403, FORBIDDEN              | P0       |

#### **Auth Requirements:**

- ✅ Admin: Any category
- ✅ Seller: Own seller categories only

---

### **1.9 Remove Attributes from Category** ⭐ NEW

`DELETE /api/categories/:id/attributes/:attribute_id`

#### **Test Scenarios:**

| #                  | Scenario                                     | Input                               | Expected Output                      | Priority |
| ------------------ | -------------------------------------------- | ----------------------------------- | ------------------------------------ | -------- |
| **Basic Removal**  |                                              |                                     |                                      |          |
| 1                  | Admin removes attribute from global category | Valid IDs, admin token              | 204, attribute removed               | P0       |
| 2                  | Seller removes attribute from own category   | Valid IDs, seller token             | 204, attribute removed               | P0       |
| 3                  | Remove non-existent attribute                | Not assigned attribute              | 404, ATTRIBUTE_NOT_ASSIGNED          | P1       |
| **Business Rules** |                                              |                                     |                                      |          |
| 4                  | Remove attribute used in products            | Attribute in use                    | 400, ATTRIBUTE_IN_USE                | P0       |
| 5                  | Force remove (admin)                         | force=true, admin token             | 204, removed + cleared from products | P2       |
| 6                  | Remove inherited attribute                   | Try removing parent's attribute     | 400, CANNOT_REMOVE_INHERITED         | P1       |
| **Authorization**  |                                              |                                     |                                      |          |
| 7                  | Seller removes from global category          | Global category, seller token       | 403, FORBIDDEN                       | P0       |
| 8                  | Seller removes from other's category         | Other seller category, seller token | 404, CATEGORY_NOT_FOUND              | P0       |
| 9                  | Unauthorized access                          | No token                            | 401, UNAUTHORIZED                    | P0       |

#### **Auth Requirements:**

- ✅ Admin: Any category
- ✅ Seller: Own seller categories only

---

## **📊 Category Test Summary**

### **Total Category Test Coverage:**

| Endpoint                  | Test Scenarios | Data Isolation Tests | Priority Breakdown           |
| ------------------------- | -------------- | -------------------- | ---------------------------- |
| 1.1 Create Category       | 33             | 7                    | P0: 18, P1: 12, P2: 3        |
| 1.2 Get All Categories    | 44             | 4                    | P0: 15, P1: 23, P2: 6        |
| 1.3 Get Category by ID    | 30             | 3                    | P0: 10, P1: 15, P2: 5        |
| 1.4 Update Category       | 50             | 6                    | P0: 20, P1: 23, P2: 7        |
| 1.5 Delete Category       | 40             | 3                    | P0: 18, P1: 10, P2: 12       |
| 1.6 Get Attributes        | 23             | 3                    | P0: 8, P1: 10, P2: 5         |
| 1.7 Get Seller Categories | 12             | 5                    | P0: 7, P1: 5                 |
| 1.8 Assign Attributes     | 14             | 3                    | P0: 8, P1: 4, P2: 2          |
| 1.9 Remove Attributes     | 9              | 2                    | P0: 5, P1: 2, P2: 2          |
| **TOTAL**                 | **255**        | **36**               | **P0: 109, P1: 104, P2: 42** |

### **Key Data Isolation Scenarios Covered:**

✅ **Global vs Seller Categories:**

- Public/Customer: See only global categories
- Seller: See global + own seller categories
- Admin: See all categories

✅ **Cross-Seller Protection:**

- Seller A cannot access Seller B's categories
- Seller A cannot modify Seller B's categories
- Seller A cannot see Seller B in listings

✅ **Hierarchy & Parent Restrictions:**

- Seller can use global as parent
- Seller cannot use other seller's category as parent
- Global cannot use seller category as parent

✅ **Authorization Layers:**

- Public: Read-only global categories
- Customer: Read-only global categories
- Seller: CRUD on own categories, read global
- Admin: Full CRUD on all categories

✅ **Attribute Isolation:**

- Sellers can only manage attributes on own categories
- Global category attributes visible to all
- Seller category attributes visible only to owner + admin

---

## **2. Product APIs**

### **2.1 Create Product**

`POST /api/products`

#### **Test Scenarios:**

| #   | Scenario                             | Input                | Expected Output              | Priority |
| --- | ------------------------------------ | -------------------- | ---------------------------- | -------- |
| 1   | Create product successfully          | Valid all fields     | 201, product created         | P0       |
| 2   | Create with minimum fields           | Only required fields | 201, product created         | P0       |
| 3   | Empty name                           | name: ""             | 400, VALIDATION_ERROR        | P0       |
| 4   | Invalid category_id                  | Non-existent UUID    | 404, CATEGORY_NOT_FOUND      | P0       |
| 5   | Negative price                       | price: -10           | 400, VALIDATION_ERROR        | P0       |
| 6   | Negative stock                       | stock: -5            | 400, VALIDATION_ERROR        | P1       |
| 7   | Name too long                        | name: 500 chars      | 400, VALIDATION_ERROR        | P1       |
| 8   | Duplicate product name (same seller) | Existing name        | 409, PRODUCT_NAME_EXISTS     | P1       |
| 9   | Unauthorized access                  | No token             | 401, UNAUTHORIZED            | P0       |
| 10  | Non-seller/admin user                | Customer token       | 403, FORBIDDEN               | P0       |
| 11  | Create with attributes               | attribute_values[]   | 201, product with attributes | P1       |
| 12  | Invalid attribute for category       | Wrong attribute ID   | 400, INVALID_ATTRIBUTE       | P1       |
| 13  | Create with images                   | image_urls[]         | 201, product with images     | P2       |
| 14  | Invalid image URL                    | Malformed URL        | 400, VALIDATION_ERROR        | P2       |

#### **Auth Requirements:**

- ✅ Seller or Admin

---

### **2.2 Get All Products**

`GET /api/products`

#### **Test Scenarios:**

| #   | Scenario                  | Query Params                                  | Expected Output                | Priority |
| --- | ------------------------- | --------------------------------------------- | ------------------------------ | -------- |
| 1   | Get all products          | -                                             | 200, paginated products        | P0       |
| 2   | Filter by category        | category_id=<uuid>                            | 200, products in category      | P0       |
| 3   | Filter by subcategories   | category_id=<uuid>&include_subcategories=true | 200, products in category tree | P1       |
| 4   | Search by name            | search=iPhone                                 | 200, matching products         | P0       |
| 5   | Price range filter        | min_price=100&max_price=500                   | 200, products in range         | P1       |
| 6   | Sort by price ascending   | sort_by=price&order=asc                       | 200, sorted products           | P1       |
| 7   | Sort by price descending  | sort_by=price&order=desc                      | 200, sorted products           | P1       |
| 8   | Sort by created date      | sort_by=created_at&order=desc                 | 200, newest first              | P1       |
| 9   | Pagination                | page=2&limit=20                               | 200, page 2 results            | P0       |
| 10  | Filter by seller          | seller_id=<uuid>                              | 200, seller's products         | P1       |
| 11  | Filter by stock status    | in_stock=true                                 | 200, available products        | P1       |
| 12  | Multiple filters combined | category + price + search                     | 200, filtered results          | P1       |
| 13  | Empty result set          | Non-matching filters                          | 200, empty array               | P2       |
| 14  | Invalid page number       | page=-1                                       | 400, VALIDATION_ERROR          | P2       |
| 15  | Public access             | -                                             | 200, success                   | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed)

---

### **2.3 Get Product by ID**

`GET /api/products/:id`

#### **Test Scenarios:**

| #   | Scenario                     | Input               | Expected Output                 | Priority |
| --- | ---------------------------- | ------------------- | ------------------------------- | -------- |
| 1   | Get existing product         | Valid UUID          | 200, product details            | P0       |
| 2   | Product not found            | Non-existent UUID   | 404, PRODUCT_NOT_FOUND          | P0       |
| 3   | Invalid UUID format          | "abc123"            | 400, INVALID_UUID               | P1       |
| 4   | Get with variants            | Valid UUID          | 200, product + variants         | P0       |
| 5   | Get with attributes          | Valid UUID          | 200, product + attributes       | P1       |
| 6   | Get with category details    | Valid UUID          | 200, product + category         | P1       |
| 7   | Get inactive product         | Inactive product ID | 404, PRODUCT_NOT_FOUND (public) | P2       |
| 8   | Get inactive product (admin) | Inactive product ID | 200, product details            | P2       |
| 9   | Public access                | -                   | 200, success                    | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed for active products)
- ✅ Admin can see inactive products

---

### **2.4 Update Product**

`PUT /api/products/:id`

#### **Test Scenarios:**

| #   | Scenario                      | Input                    | Expected Output          | Priority |
| --- | ----------------------------- | ------------------------ | ------------------------ | -------- |
| 1   | Update name                   | New name                 | 200, updated product     | P0       |
| 2   | Update description            | New description          | 200, updated product     | P0       |
| 3   | Update price                  | New price                | 200, updated product     | P0       |
| 4   | Update category               | New category_id          | 200, product moved       | P1       |
| 5   | Update stock                  | New stock value          | 200, updated stock       | P1       |
| 6   | Product not found             | Non-existent ID          | 404, PRODUCT_NOT_FOUND   | P0       |
| 7   | Duplicate name (same seller)  | Existing name            | 409, PRODUCT_NAME_EXISTS | P1       |
| 8   | Negative price                | price: -10               | 400, VALIDATION_ERROR    | P0       |
| 9   | Invalid category              | Non-existent category_id | 404, CATEGORY_NOT_FOUND  | P1       |
| 10  | Unauthorized access           | No token                 | 401, UNAUTHORIZED        | P0       |
| 11  | Update other seller's product | Different seller token   | 403, FORBIDDEN           | P0       |
| 12  | Admin updates any product     | Admin token              | 200, updated product     | P1       |
| 13  | Update attributes             | New attribute_values[]   | 200, updated attributes  | P2       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **2.5 Delete Product**

`DELETE /api/products/:id`

#### **Test Scenarios:**

| #   | Scenario                      | Input                  | Expected Output         | Priority |
| --- | ----------------------------- | ---------------------- | ----------------------- | -------- |
| 1   | Delete own product            | Valid product ID       | 204, deleted            | P0       |
| 2   | Product not found             | Non-existent ID        | 404, PRODUCT_NOT_FOUND  | P0       |
| 3   | Delete product with orders    | Product in orders      | 400, PRODUCT_HAS_ORDERS | P0       |
| 4   | Soft delete (deactivate)      | Valid ID               | 200, product inactive   | P1       |
| 5   | Unauthorized access           | No token               | 401, UNAUTHORIZED       | P0       |
| 6   | Delete other seller's product | Different seller token | 403, FORBIDDEN          | P0       |
| 7   | Admin deletes any product     | Admin token            | 204, deleted            | P1       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

## **3. Product Variant APIs**

### **3.1 Get Product Variants**

`GET /api/products/:id/variants`

#### **Test Scenarios:**

| #   | Scenario                     | Input                    | Expected Output            | Priority |
| --- | ---------------------------- | ------------------------ | -------------------------- | -------- |
| 1   | Get all variants of product  | Valid product ID         | 200, variant list          | P0       |
| 2   | Product has no variants      | Product without variants | 200, empty array           | P1       |
| 3   | Product not found            | Non-existent ID          | 404, PRODUCT_NOT_FOUND     | P0       |
| 4   | Filter by stock availability | in_stock=true            | 200, available variants    | P1       |
| 5   | Include option details       | include_options=true     | 200, variants with options | P1       |
| 6   | Public access                | -                        | 200, success               | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed)

---

### **3.2 Create Product Variant**

`POST /api/products/:id/variants`

#### **Test Scenarios:**

| #   | Scenario                     | Input                     | Expected Output                        | Priority |
| --- | ---------------------------- | ------------------------- | -------------------------------------- | -------- |
| 1   | Create variant successfully  | Valid SKU, price, options | 201, variant created                   | P0       |
| 2   | Product not found            | Non-existent product ID   | 404, PRODUCT_NOT_FOUND                 | P0       |
| 3   | Duplicate SKU                | Existing SKU              | 409, VARIANT_SKU_EXISTS                | P0       |
| 4   | Duplicate option combination | Same color+size combo     | 409, VARIANT_OPTION_COMBINATION_EXISTS | P0       |
| 5   | Empty SKU                    | sku: ""                   | 400, VALIDATION_ERROR                  | P0       |
| 6   | Negative price               | price: -10                | 400, VALIDATION_ERROR                  | P0       |
| 7   | Negative stock               | stock: -5                 | 400, VALIDATION_ERROR                  | P1       |
| 8   | Invalid option_value_id      | Non-existent option value | 404, OPTION_VALUE_NOT_FOUND            | P1       |
| 9   | Product has no options       | Product without options   | 400, PRODUCT_HAS_NO_OPTIONS            | P1       |
| 10  | Unauthorized access          | No token                  | 401, UNAUTHORIZED                      | P0       |
| 11  | Non-owner creates variant    | Different seller token    | 403, FORBIDDEN                         | P0       |
| 12  | Admin creates variant        | Admin token               | 201, variant created                   | P1       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **3.3 Get Variant by ID**

`GET /api/products/:productId/variants/:variantId`

#### **Test Scenarios:**

| #   | Scenario                          | Input                   | Expected Output           | Priority |
| --- | --------------------------------- | ----------------------- | ------------------------- | -------- |
| 1   | Get existing variant              | Valid IDs               | 200, variant details      | P0       |
| 2   | Variant not found                 | Non-existent variant ID | 404, VARIANT_NOT_FOUND    | P0       |
| 3   | Product not found                 | Non-existent product ID | 404, PRODUCT_NOT_FOUND    | P0       |
| 4   | Variant doesn't belong to product | Mismatched IDs          | 404, VARIANT_NOT_FOUND    | P1       |
| 5   | Include option details            | Valid IDs               | 200, variant with options | P1       |
| 6   | Public access                     | -                       | 200, success              | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed)

---

### **3.4 Update Product Variant**

`PUT /api/products/:productId/variants/:variantId`

#### **Test Scenarios:**

| #   | Scenario              | Input                  | Expected Output                        | Priority |
| --- | --------------------- | ---------------------- | -------------------------------------- | -------- |
| 1   | Update SKU            | New SKU                | 200, updated variant                   | P0       |
| 2   | Update price          | New price              | 200, updated variant                   | P0       |
| 3   | Update stock          | New stock value        | 200, updated variant                   | P0       |
| 4   | Variant not found     | Non-existent ID        | 404, VARIANT_NOT_FOUND                 | P0       |
| 5   | Duplicate SKU         | Existing SKU           | 409, VARIANT_SKU_EXISTS                | P0       |
| 6   | Negative price        | price: -10             | 400, VALIDATION_ERROR                  | P0       |
| 7   | Update option values  | New option_value_ids[] | 200, updated options                   | P1       |
| 8   | Duplicate combination | Existing combo         | 409, VARIANT_OPTION_COMBINATION_EXISTS | P1       |
| 9   | Unauthorized access   | No token               | 401, UNAUTHORIZED                      | P0       |
| 10  | Non-owner updates     | Different seller token | 403, FORBIDDEN                         | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **3.5 Delete Product Variant**

`DELETE /api/products/:productId/variants/:variantId`

#### **Test Scenarios:**

| #   | Scenario                    | Input                    | Expected Output                      | Priority |
| --- | --------------------------- | ------------------------ | ------------------------------------ | -------- |
| 1   | Delete variant successfully | Valid IDs                | 204, deleted                         | P0       |
| 2   | Variant not found           | Non-existent ID          | 404, VARIANT_NOT_FOUND               | P0       |
| 3   | Delete last variant         | Only remaining variant   | 400, LAST_VARIANT_DELETE_NOT_ALLOWED | P0       |
| 4   | Variant in orders           | Variant in active orders | 400, VARIANT_IN_ORDERS               | P1       |
| 5   | Unauthorized access         | No token                 | 401, UNAUTHORIZED                    | P0       |
| 6   | Non-owner deletes           | Different seller token   | 403, FORBIDDEN                       | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **3.6 Update Variant Stock**

`PATCH /api/products/:productId/variants/:variantId/stock`

#### **Test Scenarios:**

| #   | Scenario                        | Input                              | Expected Output              | Priority |
| --- | ------------------------------- | ---------------------------------- | ---------------------------- | -------- |
| 1   | Increment stock                 | operation: "add", quantity: 10     | 200, updated stock           | P0       |
| 2   | Decrement stock                 | operation: "subtract", quantity: 5 | 200, updated stock           | P0       |
| 3   | Set absolute stock              | operation: "set", quantity: 100    | 200, stock set               | P0       |
| 4   | Insufficient stock for subtract | Subtract more than available       | 400, INSUFFICIENT_STOCK      | P0       |
| 5   | Invalid operation               | operation: "invalid"               | 400, INVALID_STOCK_OPERATION | P0       |
| 6   | Negative quantity               | quantity: -10                      | 400, VALIDATION_ERROR        | P1       |
| 7   | Variant not found               | Non-existent ID                    | 404, VARIANT_NOT_FOUND       | P0       |
| 8   | Unauthorized access             | No token                           | 401, UNAUTHORIZED            | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **3.7 Bulk Update Variants**

`PUT /api/products/:id/variants/bulk`

#### **Test Scenarios:**

| #   | Scenario                 | Input                        | Expected Output                    | Priority |
| --- | ------------------------ | ---------------------------- | ---------------------------------- | -------- |
| 1   | Update multiple variants | Array of variant updates     | 200, all updated                   | P0       |
| 2   | Empty update list        | variants: []                 | 400, BULK_UPDATE_EMPTY_LIST        | P0       |
| 3   | Some variants not found  | Mix of valid/invalid IDs     | 404, BULK_UPDATE_VARIANT_NOT_FOUND | P1       |
| 4   | Partial success handling | Some updates fail validation | 207, multi-status response         | P2       |
| 5   | Unauthorized access      | No token                     | 401, UNAUTHORIZED                  | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

## **4. Product Option APIs**

### **4.1 Get Product Options**

`GET /api/products/:id/options`

#### **Test Scenarios:**

| #   | Scenario                   | Input                   | Expected Output              | Priority |
| --- | -------------------------- | ----------------------- | ---------------------------- | -------- |
| 1   | Get all options of product | Valid product ID        | 200, option list with values | P0       |
| 2   | Product has no options     | Product without options | 200, empty array             | P1       |
| 3   | Product not found          | Non-existent ID         | 404, PRODUCT_NOT_FOUND       | P0       |
| 4   | Public access              | -                       | 200, success                 | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed)

---

### **4.2 Create Product Option**

`POST /api/products/:id/options`

#### **Test Scenarios:**

| #   | Scenario                 | Input                                  | Expected Output           | Priority |
| --- | ------------------------ | -------------------------------------- | ------------------------- | -------- |
| 1   | Create option (Color)    | name: "Color", values: ["Red", "Blue"] | 201, option created       | P0       |
| 2   | Create option (Size)     | name: "Size", values: ["S", "M", "L"]  | 201, option created       | P0       |
| 3   | Empty option name        | name: ""                               | 400, VALIDATION_ERROR     | P0       |
| 4   | Empty values array       | values: []                             | 400, VALIDATION_ERROR     | P0       |
| 5   | Duplicate option name    | Existing name                          | 409, OPTION_NAME_EXISTS   | P0       |
| 6   | Product not found        | Non-existent product ID                | 404, PRODUCT_NOT_FOUND    | P0       |
| 7   | Too many options         | > max allowed                          | 400, MAX_OPTIONS_EXCEEDED | P2       |
| 8   | Unauthorized access      | No token                               | 401, UNAUTHORIZED         | P0       |
| 9   | Non-owner creates option | Different seller token                 | 403, FORBIDDEN            | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **4.3 Update Product Option**

`PUT /api/products/:productId/options/:optionId`

#### **Test Scenarios:**

| #   | Scenario                     | Input           | Expected Output          | Priority |
| --- | ---------------------------- | --------------- | ------------------------ | -------- |
| 1   | Update option name           | New name        | 200, updated option      | P0       |
| 2   | Add option value             | Add new value   | 200, value added         | P0       |
| 3   | Remove option value          | Remove value    | 200, value removed       | P1       |
| 4   | Option not found             | Non-existent ID | 404, OPTION_NOT_FOUND    | P0       |
| 5   | Duplicate name               | Existing name   | 409, OPTION_NAME_EXISTS  | P0       |
| 6   | Remove value used in variant | Value in use    | 400, OPTION_VALUE_IN_USE | P1       |
| 7   | Unauthorized access          | No token        | 401, UNAUTHORIZED        | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

### **4.4 Delete Product Option**

`DELETE /api/products/:productId/options/:optionId`

#### **Test Scenarios:**

| #   | Scenario                | Input           | Expected Output       | Priority |
| --- | ----------------------- | --------------- | --------------------- | -------- |
| 1   | Delete unused option    | Valid ID        | 204, deleted          | P0       |
| 2   | Option not found        | Non-existent ID | 404, OPTION_NOT_FOUND | P0       |
| 3   | Option used in variants | Option in use   | 400, OPTION_IN_USE    | P0       |
| 4   | Unauthorized access     | No token        | 401, UNAUTHORIZED     | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

## **5. Product Attribute APIs**

### **5.1 Get Product Attributes**

`GET /api/products/:id/attributes`

#### **Test Scenarios:**

| #   | Scenario                  | Input                      | Expected Output        | Priority |
| --- | ------------------------- | -------------------------- | ---------------------- | -------- |
| 1   | Get all attributes        | Valid product ID           | 200, attribute list    | P0       |
| 2   | Product has no attributes | Product without attributes | 200, empty array       | P1       |
| 3   | Product not found         | Non-existent ID            | 404, PRODUCT_NOT_FOUND | P0       |
| 4   | Public access             | -                          | 200, success           | P0       |

#### **Auth Requirements:**

- ✅ Public (no auth needed)

---

### **5.2 Update Product Attributes**

`PUT /api/products/:id/attributes`

#### **Test Scenarios:**

| #   | Scenario                       | Input                        | Expected Output                     | Priority |
| --- | ------------------------------ | ---------------------------- | ----------------------------------- | -------- |
| 1   | Update attribute values        | Array of attribute updates   | 200, updated attributes             | P0       |
| 2   | Invalid attribute for category | Attribute not in category    | 400, INVALID_ATTRIBUTE_FOR_CATEGORY | P0       |
| 3   | Invalid value type             | String for numeric attribute | 400, INVALID_ATTRIBUTE_VALUE        | P1       |
| 4   | Product not found              | Non-existent ID              | 404, PRODUCT_NOT_FOUND              | P0       |
| 5   | Unauthorized access            | No token                     | 401, UNAUTHORIZED                   | P0       |

#### **Auth Requirements:**

- ✅ Owner (Seller) or Admin

---

## **🔄 User Flow Tests**

### **Flow 1: Admin Category Management**

```
1. Login as Admin
   ├─> POST /api/login
   └─> Store auth token

2. Create Root Category (Electronics)
   ├─> POST /api/categories
   └─> Store category_id

3. Create Subcategory (Mobile Phones)
   ├─> POST /api/categories (parent_id = Electronics)
   └─> Store subcategory_id

4. Create Attribute (Brand)
   ├─> POST /api/attributes
   └─> Store attribute_id

5. Assign Attribute to Category
   ├─> POST /api/categories/:id/attributes
   └─> Verify assignment

6. Get Category Hierarchy
   ├─> GET /api/categories?hierarchy=true
   └─> Verify nested structure

7. Update Category
   ├─> PUT /api/categories/:id
   └─> Verify changes

8. Attempt Delete with Products (should fail)
   └─> DELETE /api/categories/:id
```

**Assertions:**

- ✅ All CRUD operations successful
- ✅ Hierarchy maintained correctly
- ✅ Attributes linked properly
- ✅ Business rules enforced (can't delete with products)

---

### **Flow 2: Seller Product Creation with Variants**

```
1. Login as Seller
   ├─> POST /api/login
   └─> Store auth token

2. Get Available Categories
   ├─> GET /api/categories
   └─> Select "Mobile Phones" category

3. Create Product (iPhone 15)
   ├─> POST /api/products
   │   {
   │     "name": "iPhone 15",
   │     "category_id": "<mobile_phones_id>",
   │     "description": "Latest iPhone",
   │     "base_price": 999.99
   │   }
   └─> Store product_id

4. Create Option: Color
   ├─> POST /api/products/:id/options
   │   {
   │     "name": "Color",
   │     "values": ["Black", "White", "Blue"]
   │   }
   └─> Store color_option_id and value_ids

5. Create Option: Storage
   ├─> POST /api/products/:id/options
   │   {
   │     "name": "Storage",
   │     "values": ["128GB", "256GB", "512GB"]
   │   }
   └─> Store storage_option_id and value_ids

6. Create Variant: Black + 128GB
   ├─> POST /api/products/:id/variants
   │   {
   │     "sku": "IPH15-BLK-128",
   │     "price": 999.99,
   │     "stock": 50,
   │     "option_values": [black_value_id, 128gb_value_id]
   │   }
   └─> Store variant_id_1

7. Create Variant: White + 256GB
   ├─> POST /api/products/:id/variants
   │   {
   │     "sku": "IPH15-WHT-256",
   │     "price": 1099.99,
   │     "stock": 30,
   │     "option_values": [white_value_id, 256gb_value_id]
   │   }
   └─> Store variant_id_2

8. Create Variant: Blue + 512GB
   ├─> POST /api/products/:id/variants
   │   {
   │     "sku": "IPH15-BLU-512",
   │     "price": 1299.99,
   │     "stock": 20,
   │     "option_values": [blue_value_id, 512gb_value_id]
   │   }
   └─> Store variant_id_3

9. Update Product Attributes
   ├─> PUT /api/products/:id/attributes
   │   {
   │     "attributes": [
   │       {"attribute_id": brand_id, "value": "Apple"},
   │       {"attribute_id": screen_size_id, "value": "6.1"}
   │     ]
   │   }
   └─> Verify updated

10. Get Product with All Details
    ├─> GET /api/products/:id
    └─> Verify: product + options + variants + attributes

11. Update Variant Stock
    ├─> PATCH /api/products/:id/variants/:variant_id/stock
    │   {
    │     "operation": "subtract",
    │     "quantity": 5
    │   }
    └─> Verify stock decreased

12. Attempt Duplicate SKU (should fail)
    ├─> POST /api/products/:id/variants
    │   {"sku": "IPH15-BLK-128", ...}
    └─> Expect 409 VARIANT_SKU_EXISTS

13. Attempt Duplicate Combination (should fail)
    ├─> POST /api/products/:id/variants
    │   {"option_values": [black_value_id, 128gb_value_id], ...}
    └─> Expect 409 VARIANT_OPTION_COMBINATION_EXISTS
```

**Assertions:**

- ✅ Product created with correct category
- ✅ Options created successfully
- ✅ Variants created with unique SKU + combinations
- ✅ Attributes assigned correctly
- ✅ Stock management works
- ✅ Duplicate validations enforced

---

### **Flow 3: Customer Product Discovery**

```
1. Browse All Products (Public)
   ├─> GET /api/products
   └─> Verify paginated response

2. Browse by Category
   ├─> GET /api/products?category_id=<mobile_phones_id>
   └─> Verify only mobile products returned

3. Search Products
   ├─> GET /api/products?search=iPhone
   └─> Verify iPhone products returned

4. Filter by Price Range
   ├─> GET /api/products?min_price=500&max_price=1500
   └─> Verify products in range

5. Sort by Price
   ├─> GET /api/products?sort_by=price&order=asc
   └─> Verify ascending order

6. View Product Details
   ├─> GET /api/products/:id
   └─> Verify product + variants + options shown

7. Check Variant Availability
   ├─> GET /api/products/:id/variants
   └─> Verify stock info for each variant

8. Select Specific Variant
   ├─> GET /api/products/:product_id/variants/:variant_id
   └─> Verify variant details (SKU, price, stock, options)

9. View Product Attributes
   ├─> GET /api/products/:id/attributes
   └─> Verify product specifications
```

**Assertions:**

- ✅ All public endpoints accessible without auth
- ✅ Search/filter/sort work correctly
- ✅ Product details complete
- ✅ Variant information accurate
- ✅ Stock availability visible

---

### **Flow 4: Admin Product Moderation**

```
1. Login as Admin
   └─> POST /api/login

2. View All Products (including inactive)
   └─> GET /api/products?include_inactive=true

3. Update Other Seller's Product
   ├─> PUT /api/products/:id
   └─> Verify admin can update any product

4. Deactivate Product
   ├─> PATCH /api/products/:id/status
   │   {"is_active": false}
   └─> Verify product hidden from public

5. Verify Public Can't See Inactive Product
   ├─> GET /api/products/:id (no auth)
   └─> Expect 404

6. Admin Can Still See Inactive Product
   ├─> GET /api/products/:id (admin auth)
   └─> Expect 200

7. Delete Product
   └─> DELETE /api/products/:id
```

**Assertions:**

- ✅ Admin has full access to all products
- ✅ Product visibility controlled by status
- ✅ Public access restricted correctly

---

### **Flow 5: Error Handling & Edge Cases**

```
1. Create Product with Invalid Data
   ├─> POST /api/products (empty name)
   └─> Expect 400 with validation details

2. Update Non-existent Product
   ├─> PUT /api/products/00000000-0000-0000-0000-000000000000
   └─> Expect 404

3. Delete Category with Products
   ├─> DELETE /api/categories/:id
   └─> Expect 400 CATEGORY_HAS_PRODUCTS

4. Create Variant for Product Without Options
   ├─> POST /api/products/:id/variants
   └─> Expect 400 PRODUCT_HAS_NO_OPTIONS

5. Delete Last Variant
   ├─> DELETE /api/products/:id/variants/:last_variant_id
   └─> Expect 400 LAST_VARIANT_DELETE_NOT_ALLOWED

6. Insufficient Stock Operation
   ├─> PATCH /api/products/:id/variants/:id/stock
   │   {"operation": "subtract", "quantity": 9999}
   └─> Expect 400 INSUFFICIENT_STOCK

7. Unauthorized Access
   ├─> POST /api/products (no token)
   └─> Expect 401

8. Forbidden Access
   ├─> PUT /api/products/:other_seller_product_id (seller token)
   └─> Expect 403
```

**Assertions:**

- ✅ All error codes correct
- ✅ Error messages descriptive
- ✅ Business rules enforced
- ✅ Security validations work

---

## **📊 Summary**

### **Total Test Coverage:**

| Module     | Endpoints | Test Scenarios | User Flows     |
| ---------- | --------- | -------------- | -------------- |
| Categories | 6         | ~55            | 1              |
| Products   | 5         | ~70            | 3              |
| Variants   | 7         | ~65            | 1 (integrated) |
| Options    | 4         | ~35            | 1 (integrated) |
| Attributes | 2         | ~15            | 1 (integrated) |
| **TOTAL**  | **24**    | **~240**       | **5**          |

### **Priority Breakdown:**

- **P0 (Critical)**: ~150 scenarios - Must pass before deployment
- **P1 (High)**: ~70 scenarios - Important business logic
- **P2 (Medium)**: ~20 scenarios - Nice to have
