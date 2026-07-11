export class Drawer {
    constructor() {
        this.element = document.getElementById('details-drawer');
        this.titleElem = document.getElementById('drawer-title');
        this.descElem = document.getElementById('drawer-description');
        this.statusBadge = document.getElementById('drawer-status-badge');
        this.impactElem = document.getElementById('drawer-impact');
        this.effortElem = document.getElementById('drawer-effort');
        this.ownerElem = document.getElementById('drawer-owner');
        this.idElem = document.getElementById('drawer-id');
        this.closeBtn = this.element.querySelector('.close-btn');

        this.closeBtn.addEventListener('click', () => this.close());
    }

    show(goal) {
        if (!goal) return;

        this.titleElem.innerText = goal.name || '';
        
        if (goal.description) {
            this.descElem.innerText = goal.description;
            this.descElem.classList.remove('muted-text');
        } else {
            this.descElem.innerText = 'No description provided.';
            this.descElem.classList.add('muted-text');
        }

        const status = goal.status || 'draft';
        this.statusBadge.innerText = status.replace('-', ' ');
        this.statusBadge.className = `badge status-${status}`;

        this.impactElem.innerText = goal.impact || 'unknown';
        this.effortElem.innerText = goal.effort || 'unknown';
        
        if (goal.member && goal.member.name) {
            this.ownerElem.innerText = goal.member.name;
            this.ownerElem.classList.remove('muted-text');
        } else {
            this.ownerElem.innerText = 'Unassigned';
            this.ownerElem.classList.add('muted-text');
        }

        this.idElem.innerText = goal.id || '';
        this.element.classList.add('open');
    }

    close() {
        this.element.classList.remove('open');
    }
}
