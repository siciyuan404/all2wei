import './Loading.css';

function Loading({ fullscreen = false, text = '加载中...' }) {
  if (fullscreen) {
    return (
      <div className="loading-fullscreen">
        <div className="loading-spinner" />
        {text && <p className="loading-text">{text}</p>}
      </div>
    );
  }

  return (
    <div className="loading-inline">
      <div className="loading-spinner" />
    </div>
  );
}

export default Loading;
