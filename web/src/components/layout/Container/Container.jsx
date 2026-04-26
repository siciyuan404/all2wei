import './Container.css';

function Container({ children, className = '', size = 'default' }) {
  return (
    <div className={`container container-${size} ${className}`}>
      {children}
    </div>
  );
}

export default Container;
