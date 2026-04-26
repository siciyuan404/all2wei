import { Link } from 'react-router-dom';
import { Button } from '../../common';
import MaterialCard from '../MaterialCard/MaterialCard';
import './MaterialGrid.css';

function MaterialGrid({ materials, onDelete, loading }) {
  if (loading) {
    return (
      <div className="material-grid-loading">
        <div className="material-grid-spinner" />
        <p>加载中...</p>
      </div>
    );
  }

  if (materials.length === 0) {
    return (
      <div className="material-grid-empty">
        <div className="material-grid-empty-icon">📚</div>
        <p>还没有学习资料</p>
        <Link to="/upload">
          <Button variant="primary">上传第一个资料</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="material-grid">
      {materials.map((material) => (
        <MaterialCard
          key={material.id}
          material={material}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}

export default MaterialGrid;
