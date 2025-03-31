from abc import ABC, abstractmethod
from typing import List, Dict, Optional, Union, Generic, TypeVar, Tuple, Any, Callable
import dataclasses
from dataclasses import dataclass
import datetime

T = TypeVar('T')
U = TypeVar('U')

# Interface example
class Repository(ABC):
    """Interface for data repositories."""
    
    @abstractmethod
    def find_by_id(self, id: int) -> Optional['Entity']:
        """Find an entity by its ID."""
        pass
    
    @abstractmethod
    def save(self, entity: 'Entity') -> 'Entity':
        """Save an entity to the repository."""
        pass
    
    @abstractmethod
    def delete(self, id: int) -> bool:
        """Delete an entity from the repository."""
        pass


# Base entity class
class Entity:
    """Base class for all entities."""
    id: int
    created_at: datetime.datetime
    updated_at: datetime.datetime
    
    def __init__(self, id: int = None):
        self.id = id
        self.created_at = datetime.datetime.now()
        self.updated_at = self.created_at
    
    def update(self) -> None:
        """Update the entity's updated_at timestamp."""
        self.updated_at = datetime.datetime.now()
    
    @property
    def age(self) -> datetime.timedelta:
        """Calculate the age of the entity."""
        return datetime.datetime.now() - self.created_at


# Complex class with generics and inheritance
class GenericRepository(Repository, Generic[T]):
    """Generic implementation of Repository."""
    
    _entities: Dict[int, T]
    _next_id: int
    _entity_type: type
    
    def __init__(self, entity_type: type):
        self._entities = {}
        self._next_id = 1
        self._entity_type = entity_type
    
    def find_by_id(self, id: int) -> Optional[T]:
        """Find an entity by its ID."""
        return self._entities.get(id)
    
    def find_all(self) -> List[T]:
        """Find all entities in the repository."""
        return list(self._entities.values())
    
    def save(self, entity: T) -> T:
        """Save an entity to the repository."""
        if not isinstance(entity, self._entity_type):
            raise TypeError(f"Expected {self._entity_type.__name__}, got {type(entity).__name__}")
        
        if not hasattr(entity, 'id') or entity.id is None:
            entity.id = self._next_id
            self._next_id += 1
        
        self._entities[entity.id] = entity
        return entity
    
    def delete(self, id: int) -> bool:
        """Delete an entity from the repository."""
        if id in self._entities:
            del self._entities[id]
            return True
        return False
    
    @classmethod
    def create_for(cls, entity_type: type) -> 'GenericRepository[Any]':
        """Factory method to create a repository for a specific entity type."""
        return cls(entity_type)


# Dataclass example
@dataclass
class User(Entity):
    """User entity."""
    username: str
    email: str
    password_hash: str
    is_active: bool = True
    roles: List[str] = dataclasses.field(default_factory=list)
    
    def __post_init__(self):
        super().__init__(getattr(self, 'id', None))
    
    def set_password(self, password: str) -> None:
        """Set the user's password."""
        # In a real application, this would hash the password
        self.password_hash = f"hashed_{password}"
        self.update()
    
    def check_password(self, password: str) -> bool:
        """Check if the provided password is correct."""
        return self.password_hash == f"hashed_{password}"
    
    @property
    def is_admin(self) -> bool:
        """Check if the user is an admin."""
        return "admin" in self.roles


# Class with nested class and complex methods
class AuthService:
    """Authentication service."""
    
    # Nested class
    class AuthResult:
        """Result of an authentication attempt."""
        
        success: bool
        user: Optional[User]
        error_message: Optional[str]
        token: Optional[str]
        
        def __init__(self, success: bool, user: Optional[User] = None, 
                     error_message: Optional[str] = None, token: Optional[str] = None):
            self.success = success
            self.user = user
            self.error_message = error_message
            self.token = token
    
    _user_repository: GenericRepository[User]
    _token_generator: Callable[[User], str]
    
    def __init__(self, user_repository: GenericRepository[User], 
                 token_generator: Callable[[User], str]):
        self._user_repository = user_repository
        self._token_generator = token_generator
    
    def authenticate(self, username: str, password: str) -> 'AuthService.AuthResult':
        """Authenticate a user with username and password."""
        # Find all users
        users = self._user_repository.find_all()
        
        # Find the user with matching username
        matching_users = [u for u in users if u.username == username]
        
        if not matching_users:
            return self.AuthResult(False, error_message="User not found")
        
        user = matching_users[0]
        
        # Check password
        if not user.check_password(password):
            return self.AuthResult(False, error_message="Invalid password")
        
        # Check if user is active
        if not user.is_active:
            return self.AuthResult(False, error_message="User is inactive")
        
        # Generate token
        token = self._token_generator(user)
        
        return self.AuthResult(True, user=user, token=token)


# Function with complex type annotations
def create_user_service(connection_string: str) -> Tuple[GenericRepository[User], AuthService]:
    """Create a user repository and authentication service."""
    # Create user repository
    user_repository = GenericRepository.create_for(User)
    
    # Create token generator
    def generate_token(user: User) -> str:
        """Generate a token for a user."""
        return f"token_{user.id}_{user.username}_{datetime.datetime.now().timestamp()}"
    
    # Create authentication service
    auth_service = AuthService(user_repository, generate_token)
    
    return user_repository, auth_service


# Function with Union type
def process_input(value: Union[str, int, List[str]]) -> Union[str, List[str]]:
    """Process different types of input."""
    if isinstance(value, str):
        return value.upper()
    elif isinstance(value, int):
        return str(value * 2)
    elif isinstance(value, list):
        return [item.upper() for item in value]
    else:
        raise TypeError(f"Unsupported type: {type(value)}")


# Higher-order function
def create_validator(predicate: Callable[[T], bool]) -> Callable[[T], bool]:
    """Create a validator function."""
    def validator(value: T) -> bool:
        """Validate a value."""
        try:
            return predicate(value)
        except Exception:
            return False
    
    return validator


# Main function to demonstrate usage
def main() -> None:
    """Main function to demonstrate the API."""
    # Create user repository and auth service
    user_repo, auth_service = create_user_service("mock://database")
    
    # Create admin user
    admin = User(username="admin", email="admin@example.com", password_hash="")
    admin.set_password("admin123")
    admin.roles.append("admin")
    user_repo.save(admin)
    
    # Create regular user
    user = User(username="user", email="user@example.com", password_hash="")
    user.set_password("user123")
    user_repo.save(user)
    
    # Authenticate admin
    admin_auth = auth_service.authenticate("admin", "admin123")
    if admin_auth.success:
        print(f"Admin authenticated with token: {admin_auth.token}")
    
    # Authenticate regular user
    user_auth = auth_service.authenticate("user", "user123")
    if user_auth.success:
        print(f"User authenticated with token: {user_auth.token}")
    
    # Process different inputs
    print(process_input("hello"))
    print(process_input(123))
    print(process_input(["hello", "world"]))


if __name__ == "__main__":
    main()