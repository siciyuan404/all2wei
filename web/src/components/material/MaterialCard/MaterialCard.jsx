import { Link } from 'react-router-dom';
import { Button } from '../../common';
import './MaterialCard.css';

function MaterialCard({ material, onDelete }) {
  const handleDelete = (e) => {
    e.preventDefault();
    e.stopPropagation();
    onDelete?.(material.id);
  };

  return (
    <div className="material-card">
      <div className="material-card-video">
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
          <div className="material-card-placeholder">无视频</div>
        )}
        {material.has_subtitle && (
          <span className="material-card-badge">字幕</span>
        )}
      </div>
      <div className="material-card-content">
        <h3 className="material-card-title">{material.title}</h3>
        <p className="material-card-desc">{material.description || '无描述'}</p>
        <div className="material-card-actions">
          <Link to={`/watch/${material.id}`} className="material-card-link">
            <Button variant="primary" size="small" fullWidth>
              开始学习
            </Button>
          </Link>
          <Button variant="ghost" size="small" onClick={handleDelete}>
            删除
          </Button>
        </div>
      </div>
    </div>
  );
}

export default MaterialCard;
