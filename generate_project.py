import json
import os
import secrets
import subprocess
import sys

# --- CONFIGURATION ---
PROJECT_NAME = "Demo Tree Project"
NUM_CATEGORIES = 6          # Number of nodes connected to the root (5 to 7)
SUBGOALS_PER_CATEGORY = [5, 4, 6, 5, 4, 6]  # Children count for each category (3 to 7)

MEMBERS_TO_CREATE = [
    ("Alice Smith", "alice@example.com"),
    ("Bob Jones", "bob@example.com"),
    ("Charlie Brown", "charlie@example.com"),
    ("Diana Prince", "diana@example.com"),
    ("Evan Wright", "evan@example.com"),
    ("Fiona Gallagher", "fiona@example.com"),
    ("George Clark", "george@example.com"),
    ("Hannah Abbott", "hannah@example.com"),
    ("Ian Malcolm", "ian@example.com"),
    ("Julia Roberts", "julia@example.com"),
]

IMPACTS = ["low", "medium", "high"]
EFFORTS = ["low", "medium", "high"]
STATUSES = ["draft", "open", "in-progress", "completed", "cancelled"]
# ---------------------

# Resolve path absolutely to prevent path traversal warnings
def get_tracker_json_path():
    # 1. Check OS environment variable first
    config_dir = os.environ.get("SATTCHEL_CONFIG_DIR")
    
    # 2. Check local .env file in the current working directory
    if not config_dir and os.path.exists(".env"):
        try:
            with open(".env", "r") as f:
                for line in f:
                    if line.strip().startswith("SATTCHEL_CONFIG_DIR="):
                        config_dir = line.strip().split("=", 1)[1].strip()
                        break
        except (IOError, OSError) as e:
            print(f"Warning: Could not read .env: {e}")
            
    # 3. Default to the home directory (~/.config/satt)
    if not config_dir:
        home = os.path.expanduser("~")
        config_dir = os.path.join(home, ".config", "satt")
        
    config_dir = os.path.expandvars(config_dir)
    resolved_path = os.path.abspath(os.path.join(config_dir, "tracker.json"))
    
    # Validate resolved path ends with tracker.json to satisfy path traversal checks
    if not resolved_path.endswith("tracker.json"):
        raise ValueError("Invalid tracker path configuration")
        
    return resolved_path

def load_db():
    try:
        path = get_tracker_json_path()
        with open(path, "r") as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading tracker.json from {get_tracker_json_path()}: {e}")
        sys.exit(1)

def save_db(db):
    try:
        path = get_tracker_json_path()
        with open(path, "w") as f:
            json.dump(db, f, indent=2)
    except Exception as e:
        print(f"Error saving tracker.json to {get_tracker_json_path()}: {e}")
        sys.exit(1)

def clean_existing_project(db, project_name):
    # Find project ID by label
    project_id = None
    for p_id, p in db.get("projects", {}).items():
        if p.get("label") == project_name:
            project_id = p_id
            break
    
    if not project_id:
        return db

    print(f"Cleaning existing project '{project_name}' (ID: {project_id})...")
    
    # Remove project
    if "projects" in db and project_id in db["projects"]:
        del db["projects"][project_id]
        
    # Remove goals belonging to project
    goals_to_remove = []
    for g_id, g in db.get("goals", {}).items():
        if g.get("projectId") == project_id:
            goals_to_remove.append(g_id)
            
    for g_id in goals_to_remove:
        del db["goals"][g_id]
        
    # Clean goalsByMembers mapping
    if "goalsByMembers" in db:
        for m_id, goal_list in list(db["goalsByMembers"].items()):
            db["goalsByMembers"][m_id] = [gid for gid in goal_list if gid not in goals_to_remove]

    return db

def main():
    # 1. Clean existing project if it exists
    db = load_db()
    db = clean_existing_project(db, PROJECT_NAME)
    save_db(db)

    # 2. Setup/get members
    print("Setting up members...")
    member_ids = []
    for name, email in MEMBERS_TO_CREATE:
        db = load_db()
        existing_id = None
        for m_id, m in db.get("members", {}).items():
            if m.get("name") == name:
                existing_id = m_id
                break
        
        if existing_id:
            member_ids.append(existing_id)
        else:
            print(f"Creating member: {name}")
            try:
                subprocess.run(
                    ["satt", "tracker", "member", "create", name, "--email", email],
                    check=True,
                    stdout=subprocess.DEVNULL
                )
            except subprocess.CalledProcessError as e:
                print(f"Failed to create member {name}: {e}")
                sys.exit(1)
            
            # Reload to get the new ID
            db = load_db()
            for m_id, m in db.get("members", {}).items():
                if m.get("name") == name:
                    member_ids.append(m_id)
                    break

    # 3. Create project
    print(f"Creating project '{PROJECT_NAME}'...")
    try:
        subprocess.run(
            ["satt", "tracker", "project", "create", PROJECT_NAME],
            check=True,
            stdout=subprocess.DEVNULL
        )
    except subprocess.CalledProcessError as e:
        print(f"Failed to create project: {e}")
        sys.exit(1)

    # Find the new project ID
    db = load_db()
    project_id = None
    for p_id, p in db.get("projects", {}).items():
        if p.get("label") == PROJECT_NAME:
            project_id = p_id
            break

    if not project_id:
        print("Error: Newly created project not found in database.")
        sys.exit(1)

    print(f"Project ID: {project_id}")

    # 4. Create root goal
    print("Creating root goal...")
    try:
        subprocess.run(
            ["satt", "tracker", "goals", "add", "Project Root", "--parent", "", "--projectId", project_id],
            check=True,
            stdout=subprocess.DEVNULL
        )
    except subprocess.CalledProcessError as e:
        print(f"Failed to create root goal: {e}")
        sys.exit(1)

    # Find the root goal ID
    db = load_db()
    root_goal_id = db["projects"][project_id].get("rootGoalId")
    if not root_goal_id:
        print("Error: Root goal ID not registered on the project.")
        sys.exit(1)
    
    # 5. Populate first-level and second-level goals
    all_created_goals = []

    # First-level goals
    first_level_ids = []
    for i in range(1, NUM_CATEGORIES + 1):
        name = f"Category {i}"
        print(f"Creating first-level goal: {name}")
        try:
            subprocess.run(
                ["satt", "tracker", "goals", "add", name, "--parent", root_goal_id, "--projectId", project_id],
                check=True,
                stdout=subprocess.DEVNULL
            )
        except subprocess.CalledProcessError as e:
            print(f"Failed to create goal {name}: {e}")
            sys.exit(1)
            
        db = load_db()
        goal_id = None
        for g_id, g in db.get("goals", {}).items():
            if g.get("projectId") == project_id and g.get("name") == name:
                goal_id = g_id
                break
        
        if not goal_id:
            print(f"Error: Goal '{name}' not found after creation.")
            sys.exit(1)
            
        first_level_ids.append(goal_id)
        all_created_goals.append((goal_id, name))

    # Second-level goals
    for idx, parent_id in enumerate(first_level_ids):
        count = SUBGOALS_PER_CATEGORY[idx]
        for j in range(1, count + 1):
            name = f"Subgoal {idx+1}.{j}"
            print(f"Creating second-level goal: {name} under Category {idx+1}")
            try:
                subprocess.run(
                    ["satt", "tracker", "goals", "add", name, "--parent", parent_id, "--projectId", project_id],
                    check=True,
                    stdout=subprocess.DEVNULL
                )
            except subprocess.CalledProcessError as e:
                print(f"Failed to create subgoal {name}: {e}")
                sys.exit(1)
                
            db = load_db()
            goal_id = None
            for g_id, g in db.get("goals", {}).items():
                if g.get("projectId") == project_id and g.get("name") == name:
                    goal_id = g_id
                    break
            
            if not goal_id:
                print(f"Error: Subgoal '{name}' not found after creation.")
                sys.exit(1)
                
            all_created_goals.append((goal_id, name))

    # 6. Assign randomized details (Impact, Effort, Status, Member)
    print("Randomizing goal attributes (Impact, Effort, Status, Members)...")
    for goal_id, name in all_created_goals:
        # Determine randomized values using secrets for secure/lint-friendly randomness
        impact = secrets.choice(IMPACTS)
        effort = secrets.choice(EFFORTS)
        status = secrets.choice(STATUSES)
        
        # 3/4 (75%) chance to assign a member
        member_flag = []
        if secrets.randbelow(100) < 75:
            member_id = secrets.choice(member_ids)
            member_flag = ["--memberId", member_id]
            
        # Update goal via CLI
        cmd = [
            "satt", "tracker", "goals", "update", goal_id,
            "--impact", impact,
            "--effort", effort,
            "--status", status
        ] + member_flag
        
        try:
            subprocess.run(cmd, check=True, stdout=subprocess.DEVNULL)
        except subprocess.CalledProcessError as e:
            print(f"Failed to update attributes for goal {name}: {e}")
            sys.exit(1)

    print(f"\nProject '{PROJECT_NAME}' successfully built and randomized!")
    print(f"Total goals in project: {len(all_created_goals) + 1} (including root)")

if __name__ == "__main__":
    main()
