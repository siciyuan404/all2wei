import Header from '../Header/Header';
import Container from '../Container/Container';
import './PageLayout.css';

function PageLayout({ 
  title, 
  showBack = false, 
  backTo = '/', 
  actions,
  containerSize = 'default',
  children,
  className = '',
}) {
  return (
    <div className={`page-layout ${className}`}>
      <Header title={title} showBack={showBack} backTo={backTo} actions={actions} />
      <Container size={containerSize}>
        {children}
      </Container>
    </div>
  );
}

export default PageLayout;
