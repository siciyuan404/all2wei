import { Link } from 'react-router-dom';
import { useAuth } from '../../../context/AuthContext';
import './Header.css';

function Header({ title, showBack = false, backTo = '/', actions }) {
  const { user, logout } = useAuth();

  return (
    <header className="header">
      <div className="header-left">
        {showBack && (
          <Link to={backTo} className="header-back">
            ← 返回
          </Link>
        )}
        <h1 className="header-title">{title}</h1>
      </div>
      <div className="header-right">
        {actions}
        {user && (
          <div className="header-user">
            <span className="header-username">{user.username}</span>
            <button className="header-logout" onClick={logout}>
              退出
            </button>
          </div>
        )}
      </div>
    </header>
  );
}

export default Header;
