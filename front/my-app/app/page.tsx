// ✅ 默认是服务器组件（Server Component），可以直接在服务器获取数据
// app/page.tsx

// 模拟数据获取（真实场景可以是 Prisma、axios、Drizzle 等）
async function fetchUsers() {
  // 模拟服务器延迟
  await new Promise(resolve => setTimeout(resolve, 500));

  // 返回模拟用户数据
  return [
    { id: 1, name: "张三", email: "zhangsan@example.com", role: "Admin" },
    { id: 2, name: "李四", email: "lisi@example.com", role: "User" },
    { id: 3, name: "王五", email: "wangwu@example.com", role: "User" },
  ];
}

export default async function UsersPage() {
  // ✅ 服务器组件可以直接使用 await 获取数据（无需 useEffect！）
  const users = await fetchUsers();

  return (
    <div className="container mx-auto p-8">
      <h1 className="text-4xl font-bold text-center mb-8">用户列表</h1>

      {/* 使用 Tailwind CSS 构建响应式表格 */}
      <div className="overflow-x-auto shadow-md rounded-lg">
        <table className="min-w-full bg-white border-collapse">
          <thead>
            <tr className="bg-gray-100 border-b">
              <th className="py-3 px-6 text-left text-sm font-semibold text-gray-600">ID</th>
              <th className="py-3 px-6 text-left text-sm font-semibold text-gray-600">姓名</th>
              <th className="py-3 px-6 text-left text-sm font-semibold text-gray-600">邮箱</th>
              <th className="py-3 px-6 text-left text-sm font-semibold text-gray-600">角色</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id} className="border-b hover:bg-gray-50 transition-colors">
                <td className="py-4 px-6 text-sm text-gray-900">{user.id}</td>
                <td className="py-4 px-6 text-sm text-gray-900">{user.name}</td>
                <td className="py-4 px-6 text-sm text-blue-600 underline">{user.email}</td>
                <td className="py-4 px-6">
                  <span className="px-3 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-800">
                    {user.role}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
