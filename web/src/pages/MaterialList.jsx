import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { getMaterials, deleteMaterial, syncMaterials } from '../api/material';

function MaterialList() {
  const [materials, setMaterials] = useState([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const user = JSON.parse(localStorage.getItem('user') || '{}');

  useEffect(() => {
    fetchMaterials();
  }, []);

  const fetchMaterials = async () => {
    try {
      const response = await getMaterials();
      setMaterials(response.data || []);
    } catch (err) {
      setError('获取资料列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    setError('');
    try {
      const response = await syncMaterials();
      alert(response.data.message);
      fetchMaterials(); // 刷新列表
    } catch (err) {
      setError(err.response?.data?.error || '同步失败');
    } finally {
      setSyncing(false);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('确定要删除这个学习资料吗？')) return;

    try {
      await deleteMaterial(id);
      setMaterials(materials.filter((m) => m.id !== id));
    } catch (err) {
      alert('删除失败');
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="container">
      <header className="header">
        <h1>我的学习资料</h1>
        <div className="header-actions">
          <span className="username">{user.username}</span>
          <button className="btn-logout" onClick={handleLogout}>
            退出
          </button>
        </div>
      </header>

      <div className="toolbar">
        <Link to="/upload" className="btn-primary">
          + 上传新资料
        </Link>
        <button 
          className="btn-secondary" 
          onClick={handleSync}
          disabled={syncing}
        >
          {syncing ? '同步中...' : '同步 MinIO'}
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}

      {materials.length === 0 ? (
        <div className="empty-state">
          <p>还没有学习资料</p>
          <Link to="/upload" className="btn-primary">
            上传第一个资料
          </Link>
        </div>
      ) : (
        <div className="material-grid">
          {materials.map((material) => (
            <div key={material.id} className="material-card">
              <div className="card-video">
                {material.video_url ? (
                  <video
                    src={material.video_url}
                    preload="metadata"
                    muted
                    onMouseEnter={(e) => {
                      if (e.target.src) e.target.play().catch(() => {});
                    }}
                    onMouseLeave={(e) => {
                      e.target.pause();
                      e.target.currentTime = 0;
                    }}
                  />
                ) : (
                  <div className="video-placeholder">无视频</div>
                )}
                {material.has_subtitle && (
                  <span className="subtitle-badge">字幕</span>
                )}
              </div>
              <div className="card-info">
                <h3>{material.title}</h3>
                <p className="description">{material.description || '无描述'}</p>
                <div className="card-actions">
                  <Link
                    to={`/watch/${material.id}`}
                    className="btn-watch"
                  >
                    开始学习
                  </Link>
                  <button
                    className="btn-delete"
                    onClick={() => handleDelete(material.id)}
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default MaterialList;
