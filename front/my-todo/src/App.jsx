import { useState, useEffect } from 'react';
// Import Tailwind CSS (already included via index.css)

function App() {
  // --------------------------
  // React State (核心概念：组件的局部变量)
  // --------------------------
  // 1. 待办事项列表 (Array of Objects)
  // 使用useState初始化空数组
  const [todos, setTodos] = useState([]);

  // 2. 输入框的值 (Single String)
  // 使用useState初始化空字符串
  const [inputValue, setInputValue] = useState('');

  // --------------------------
  // React useEffect (核心概念：副作用处理)
  // --------------------------
  // 1. 基础使用：每次组件渲染后执行
  // 用途：调试、日志记录等
  useEffect(() => {
    console.log('组件已渲染 - Basic useEffect');
  });

  // 2. 仅在组件首次挂载时执行（依赖项为空数组）
  // 用途：初始化数据、订阅事件、DOM操作等
  useEffect(() => {
    console.log('组件首次挂载 - useEffect with empty dependencies');
    // 示例：模拟从API获取初始待办事项
    // setTimeout(() => {
    //   setTodos([{ id: 1, text: '学习React' }, { id: 2, text: '掌握useEffect' }]);
    // }, 1000);
  }, []);

  // 3. 当特定依赖项变化时执行
  // 用途：响应特定state或props的变化
  useEffect(() => {
    console.log('待办事项列表已更新:', todos);
  }, [todos]);

  // 4. 清理函数：当组件卸载或依赖项变化前执行
  // 用途：取消订阅、清除定时器、清理DOM等
  useEffect(() => {
    const timer = setTimeout(() => {
      console.log('3秒后执行的定时器');
    }, 3000);

    // 清理函数
    return () => {
      clearTimeout(timer);
      console.log('定时器已清理');
    };
  }, []);

  // 5. 多个依赖项：当任意依赖项变化时执行
  // 用途：根据多个state或props的变化执行操作
  useEffect(() => {
    if (inputValue.length > 10) {
      console.log('输入内容已超过10个字符');
    }
  }, [inputValue, todos]); // 监听inputValue和todos的变化

  // --------------------------
  // Event Handlers (事件处理函数)
  // --------------------------
  // 1. 输入框变化时更新state
  const handleInputChange = (e) => {
    // e.target.value 是输入框当前的值
    setInputValue(e.target.value);
  };

  // 2. 表单提交时添加待办事项
  const handleFormSubmit = (e) => {
    // 阻止表单默认提交行为（避免页面刷新）
    e.preventDefault();

    // 非空验证
    if (inputValue.trim() === '') return;

    // 创建新的待办事项对象，包含id和内容
    const newTodo = {
      id: Date.now(), // 使用当前时间作为唯一id（简单方案）
      text: inputValue.trim()
    };

    // 更新待办事项列表：[...旧列表, 新列表项]
    setTodos([...todos, newTodo]);

    // 清空输入框
    setInputValue('');
  };

  // 3. 删除待办事项
  const handleDeleteTodo = (id) => {
    // 过滤掉id匹配的待办事项
    const newTodos = todos.filter(todo => todo.id !== id);
    setTodos(newTodos);
  };

  // --------------------------
  // Render JSX (渲染UI)
  // --------------------------
  return (
    <div className="max-w-4xl mx-auto p-8">
      <h1 className="text-4xl font-bold text-center mb-8 text-gray-800">TODO List</h1>

      {/* 表单：添加待办事项 */}
      <form onSubmit={handleFormSubmit} className="flex gap-2 mb-8">
        <input
          type="text"
          placeholder="添加新的待办事项..."
          value={inputValue}
          onChange={handleInputChange}
          className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          type="submit"
          className="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
        >
          添加
        </button>
      </form>

      {/* 待办事项列表 */}
      <div className="space-y-3">
        {todos.length === 0 ? (
          <p className="text-center text-gray-500">暂无待办事项</p>
        ) : (
          todos.map(todo => (
            <div
              key={todo.id}
              className="flex justify-between items-center p-4 bg-white border border-gray-200 rounded-lg shadow-sm"
            >
              <span className="text-gray-700">{todo.text}</span>
              <button
                onClick={() => handleDeleteTodo(todo.id)}
                className="px-4 py-1 bg-red-500 text-white rounded-md hover:bg-red-600 transition-colors"
              >
                删除
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

export default App;
